// Package revenue proxies requests to the Massachusetts Socrata revenue API.
package revenue

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"

	sc "massdata/api/socrata"
	"massdata/models"
)

const viewID = "kcy7-ivxi"

const selectCols = "`fiscal_year`, `fiscal_period`, `fiscal_period_month`, `apd_nm`," +
	" `revenue_category_name`, `fund_category_name`, `fund_number`, `fund_name`," +
	" `sub_fund`, `sub_fund_name`, `revenue_class_name`, `cabinet_name`," +
	" `department_code`, `department_name`, `revenue_source`, `revenue_source_name`," +
	" `revenue_collected`, `trans_no`"

// Handler serves GET /api/mass-revenue.
type Handler struct {
	client *sc.Client
}

// NewHandler creates a revenue handler with the given Socrata app token.
func NewHandler(appToken string, devMode bool) *Handler {
	return &Handler{client: sc.NewClient(appToken, devMode)}
}

type socrataRaw struct {
	FiscalYear          string `json:"fiscal_year"`
	FiscalPeriod        string `json:"fiscal_period"`
	FiscalPeriodMonth   string `json:"fiscal_period_month"`
	ApdNm               string `json:"apd_nm"`
	RevenueCategoryName string `json:"revenue_category_name"`
	FundCategoryName    string `json:"fund_category_name"`
	FundNumber          string `json:"fund_number"`
	FundName            string `json:"fund_name"`
	SubFund             string `json:"sub_fund"`
	SubFundName         string `json:"sub_fund_name"`
	RevenueClassName    string `json:"revenue_class_name"`
	CabinetName         string `json:"cabinet_name"`
	DepartmentCode      string `json:"department_code"`
	DepartmentName      string `json:"department_name"`
	RevenueSource       string `json:"revenue_source"`
	RevenueSourceName   string `json:"revenue_source_name"`
	RevenueCollected    string `json:"revenue_collected"`
	TransNo             string `json:"trans_no"`
}

// GetRevenueData handles GET /api/mass-revenue.
// Supported query params: year, page, limit, sort, sort_field,
// dept, cabinet, category, class, fund.
func (h *Handler) GetRevenueData(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	year := q.Get("year")
	if year == "" {
		year = "2024"
	}
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 {
		limit = 25
	}
	sortDir := "DESC"
	if q.Get("sort") == "asc" {
		sortDir = "ASC"
	}
	sortCol := mapSortField(q.Get("sort_field"))

	dept := sc.SanitizeSQL(q.Get("dept"))
	cabinet := sc.SanitizeSQL(q.Get("cabinet"))
	category := sc.SanitizeSQL(q.Get("category"))
	class := sc.SanitizeSQL(q.Get("class"))
	fund := sc.SanitizeSQL(q.Get("fund"))

	where := buildWhere(sc.SanitizeSQL(year), dept, cabinet, category, class, fund)

	dataSQL := fmt.Sprintf(
		"SELECT %s %s ORDER BY `%s` %s LIMIT %d OFFSET %d",
		selectCols, where, sortCol, sortDir, limit, page*limit,
	)
	aggSQL := fmt.Sprintf(
		"SELECT COUNT(*) as total_count, SUM(`revenue_collected`) as total_amount %s",
		where,
	)

	var (
		rawData json.RawMessage
		rawAgg  json.RawMessage
		dataErr error
		aggErr  error
		wg      sync.WaitGroup
	)
	wg.Add(2)
	go func() { defer wg.Done(); rawData, dataErr = h.client.Query(viewID, dataSQL) }()
	go func() { defer wg.Done(); rawAgg, aggErr = h.client.Query(viewID, aggSQL) }()
	wg.Wait()

	if dataErr != nil {
		log.Printf("[ERROR] revenue data query: %v", dataErr)
		sc.WriteError(w, "failed to fetch revenue data", http.StatusBadGateway)
		return
	}
	if aggErr != nil {
		log.Printf("[ERROR] revenue aggregate query: %v", aggErr)
		sc.WriteError(w, "failed to fetch revenue aggregates", http.StatusBadGateway)
		return
	}

	var raws []socrataRaw
	if err := json.Unmarshal(rawData, &raws); err != nil {
		log.Printf("[ERROR] revenue parse records: %v", err)
		sc.WriteError(w, "failed to parse revenue data", http.StatusInternalServerError)
		return
	}

	records := make([]models.RevenueRecord, 0, len(raws))
	for _, rec := range raws {
		collected, _ := strconv.ParseFloat(rec.RevenueCollected, 64)
		records = append(records, models.RevenueRecord{
			FiscalYear:          rec.FiscalYear,
			FiscalPeriod:        rec.FiscalPeriod,
			FiscalPeriodMonth:   rec.FiscalPeriodMonth,
			ApdNm:               rec.ApdNm,
			RevenueCategoryName: rec.RevenueCategoryName,
			FundCategoryName:    rec.FundCategoryName,
			FundNumber:          rec.FundNumber,
			FundName:            rec.FundName,
			SubFund:             rec.SubFund,
			SubFundName:         rec.SubFundName,
			RevenueClassName:    rec.RevenueClassName,
			CabinetName:         rec.CabinetName,
			DepartmentCode:      rec.DepartmentCode,
			DepartmentName:      rec.DepartmentName,
			RevenueSource:       rec.RevenueSource,
			RevenueSourceName:   rec.RevenueSourceName,
			RevenueCollected:    collected,
			TransNo:             rec.TransNo,
		})
	}

	var aggs []sc.Aggregate
	if err := json.Unmarshal(rawAgg, &aggs); err != nil {
		log.Printf("[ERROR] revenue parse aggregates: %v", err)
		sc.WriteError(w, "failed to parse revenue aggregates", http.StatusInternalServerError)
		return
	}

	var count int
	var totalAmount float64
	if len(aggs) > 0 {
		count = int(sc.ParseNumber(aggs[0].Count))
		totalAmount = sc.ParseNumber(aggs[0].TotalAmount)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.RevenueResponse{
		Data:        records,
		Count:       count,
		TotalAmount: totalAmount,
	})
}

func buildWhere(year, dept, cabinet, category, class, fund string) string {
	conds := []string{fmt.Sprintf("`fiscal_year` = '%s'", year)}
	if dept != "" {
		conds = append(conds, fmt.Sprintf("UPPER(`department_name`) LIKE UPPER('%%%s%%')", dept))
	}
	if cabinet != "" {
		conds = append(conds, fmt.Sprintf("UPPER(`cabinet_name`) LIKE UPPER('%%%s%%')", cabinet))
	}
	if category != "" {
		conds = append(conds, fmt.Sprintf("UPPER(`revenue_category_name`) LIKE UPPER('%%%s%%')", category))
	}
	if class != "" {
		conds = append(conds, fmt.Sprintf("UPPER(`revenue_class_name`) LIKE UPPER('%%%s%%')", class))
	}
	if fund != "" {
		conds = append(conds, fmt.Sprintf("UPPER(`fund_name`) LIKE UPPER('%%%s%%')", fund))
	}
	return "WHERE " + strings.Join(conds, " AND ")
}

// TODO: review this mapping and remove if not needed, as it doesn't
// appear to be mapping anything.
func mapSortField(f string) string {
	switch f {
	case "department_name":
		return "department_name"
	case "cabinet_name":
		return "cabinet_name"
	case "revenue_category_name":
		return "revenue_category_name"
	case "revenue_class_name":
		return "revenue_class_name"
	case "revenue_source_name":
		return "revenue_source_name"
	case "fund_name":
		return "fund_name"
	case "revenue_collected":
		return "revenue_collected"
	case "fiscal_period":
		return "fiscal_period"
	case "fiscal_year":
		return "fiscal_year"
	default:
		return "fiscal_year"
	}
}
