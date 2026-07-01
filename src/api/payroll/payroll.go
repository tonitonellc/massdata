// Package payroll proxies requests to the Massachusetts Socrata payroll API.
package payroll

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

const viewID = "9ttk-7vz6"

const selectCols = "`id`, `year`, `trans_no`, `name_last`, `name_first`," +
	" `department_division`, `department_code`, `position_title`, `position_type`," +
	" `service_end_date`, `pay_total_actual`, `pay_base_actual`, `pay_buyout_actual`," +
	" `pay_overtime_actual`, `pay_other_actual`, `annual_rate`, `pay_year_to_date`," +
	" `department_location_zip_code`, `contract`, `bargaining_group_no`, `bargaining_group_title`"

// Handler serves GET /api/mass-payroll.
type Handler struct {
	client *sc.Client
}

// NewHandler creates a payroll handler with the given Socrata app token.
func NewHandler(appToken string, devMode bool) *Handler {
	return &Handler{client: sc.NewClient(appToken, devMode)}
}

type socrataRaw struct {
	ID                   string `json:"id"`
	Year                 string `json:"year"`
	TransNo              string `json:"trans_no"`
	NameLast             string `json:"name_last"`
	NameFirst            string `json:"name_first"`
	DepartmentDivision   string `json:"department_division"`
	DepartmentCode       string `json:"department_code"`
	PositionTitle        string `json:"position_title"`
	PositionType         string `json:"position_type"`
	ServiceEndDate       string `json:"service_end_date"`
	PayTotalActual       string `json:"pay_total_actual"`
	PayBaseActual        string `json:"pay_base_actual"`
	PayBuyoutActual      string `json:"pay_buyout_actual"`
	PayOvertimeActual    string `json:"pay_overtime_actual"`
	PayOtherActual       string `json:"pay_other_actual"`
	AnnualRate           string `json:"annual_rate"`
	PayYearToDate        string `json:"pay_year_to_date"`
	DeptLocationZipCode  string `json:"department_location_zip_code"`
	Contract             string `json:"contract"`
	BargainingGroupNo    string `json:"bargaining_group_no"`
	BargainingGroupTitle string `json:"bargaining_group_title"`
}

// GetPayrollData handles GET /api/mass-payroll.
// Supported query params: year, page, limit, sort, sort_field,
// last, first, dept, title, type, bargaining.
func (h *Handler) GetPayrollData(w http.ResponseWriter, r *http.Request) {
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

	last := sc.SanitizeSQL(q.Get("last"))
	first := sc.SanitizeSQL(q.Get("first"))
	dept := sc.SanitizeSQL(q.Get("dept"))
	title := sc.SanitizeSQL(q.Get("title"))
	posType := sc.SanitizeSQL(q.Get("type"))
	bargaining := sc.SanitizeSQL(q.Get("bargaining"))

	where := buildWhere(sc.SanitizeSQL(year), last, first, dept, title, posType, bargaining)

	dataSQL := fmt.Sprintf(
		"SELECT %s %s ORDER BY `%s` %s LIMIT %d OFFSET %d",
		selectCols, where, sortCol, sortDir, limit, page*limit,
	)
	aggSQL := fmt.Sprintf(
		"SELECT COUNT(*) as total_count, SUM(`pay_total_actual`) as total_amount %s",
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
		log.Printf("[ERROR] payroll data query: %v", dataErr)
		sc.WriteError(w, "failed to fetch payroll data", http.StatusBadGateway)
		return
	}
	if aggErr != nil {
		log.Printf("[ERROR] payroll aggregate query: %v", aggErr)
		sc.WriteError(w, "failed to fetch payroll aggregates", http.StatusBadGateway)
		return
	}

	var raws []socrataRaw
	if err := json.Unmarshal(rawData, &raws); err != nil {
		log.Printf("[ERROR] payroll parse records: %v", err)
		sc.WriteError(w, "failed to parse payroll data", http.StatusInternalServerError)
		return
	}

	parseFloat := func(s string) float64 { f, _ := strconv.ParseFloat(s, 64); return f }

	records := make([]models.PayrollRecord, 0, len(raws))
	for _, rec := range raws {
		records = append(records, models.PayrollRecord{
			ID:                   rec.ID,
			Year:                 rec.Year,
			TransNo:              rec.TransNo,
			NameLast:             rec.NameLast,
			NameFirst:            rec.NameFirst,
			DepartmentDivision:   rec.DepartmentDivision,
			DepartmentCode:       rec.DepartmentCode,
			PositionTitle:        rec.PositionTitle,
			PositionType:         rec.PositionType,
			ServiceEndDate:       rec.ServiceEndDate,
			PayTotalActual:       parseFloat(rec.PayTotalActual),
			PayBaseActual:        parseFloat(rec.PayBaseActual),
			PayBuyoutActual:      parseFloat(rec.PayBuyoutActual),
			PayOvertimeActual:    parseFloat(rec.PayOvertimeActual),
			PayOtherActual:       parseFloat(rec.PayOtherActual),
			AnnualRate:           parseFloat(rec.AnnualRate),
			PayYearToDate:        parseFloat(rec.PayYearToDate),
			DeptLocationZipCode:  rec.DeptLocationZipCode,
			Contract:             rec.Contract,
			BargainingGroupNo:    rec.BargainingGroupNo,
			BargainingGroupTitle: rec.BargainingGroupTitle,
		})
	}

	var aggs []sc.Aggregate
	if err := json.Unmarshal(rawAgg, &aggs); err != nil {
		log.Printf("[ERROR] payroll parse aggregates: %v", err)
		sc.WriteError(w, "failed to parse payroll aggregates", http.StatusInternalServerError)
		return
	}

	var count int
	var totalAmount float64
	if len(aggs) > 0 {
		count = int(sc.ParseNumber(aggs[0].Count))
		totalAmount = sc.ParseNumber(aggs[0].TotalAmount)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.PayrollResponse{
		Data:        records,
		Count:       count,
		TotalAmount: totalAmount,
	})
}

func buildWhere(year, last, first, dept, title, posType, bargaining string) string {
	conds := []string{fmt.Sprintf("`year` = '%s'", year)}
	if last != "" {
		conds = append(conds, fmt.Sprintf("UPPER(`name_last`) LIKE UPPER('%%%s%%')", last))
	}
	if first != "" {
		conds = append(conds, fmt.Sprintf("UPPER(`name_first`) LIKE UPPER('%%%s%%')", first))
	}
	if dept != "" {
		conds = append(conds, fmt.Sprintf("UPPER(`department_division`) LIKE UPPER('%%%s%%')", dept))
	}
	if title != "" {
		conds = append(conds, fmt.Sprintf("UPPER(`position_title`) LIKE UPPER('%%%s%%')", title))
	}
	if posType != "" {
		conds = append(conds, fmt.Sprintf("UPPER(`position_type`) LIKE UPPER('%%%s%%')", posType))
	}
	if bargaining != "" {
		conds = append(conds, fmt.Sprintf("UPPER(`bargaining_group_title`) LIKE UPPER('%%%s%%')", bargaining))
	}
	return "WHERE " + strings.Join(conds, " AND ")
}

// TODO: review this mapping and remove if not needed, as it doesn't
// appear to be mapping anything.
func mapSortField(f string) string {
	switch f {
	case "name_last":
		return "name_last"
	case "name_first":
		return "name_first"
	case "department_division":
		return "department_division"
	case "position_title":
		return "position_title"
	case "position_type":
		return "position_type"
	case "pay_total_actual":
		return "pay_total_actual"
	case "pay_base_actual":
		return "pay_base_actual"
	case "pay_overtime_actual":
		return "pay_overtime_actual"
	case "annual_rate":
		return "annual_rate"
	case "service_end_date":
		return "service_end_date"
	case "year":
		return "year"
	case "bargaining_group_title":
		return "bargaining_group_title"
	default:
		return "service_end_date"
	}
}
