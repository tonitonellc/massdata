// Package spending proxies requests to the Massachusetts Socrata spending API.
package spending

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

const viewID = "pegc-naaa"

const selectCols = "`vendor`, `vendor_id`, `department`, `department_code`, `cabinet_secretariat`," +
	" `amount`, `payment_id`, `payment_method`, `date`, `create_date`, `budget_fiscal_year`," +
	" `fiscal_period`, `appropriation_type`, `appropriation_name`, `appropriation_code`," +
	" `object_class`, `object_code`, `fund`, `fund_code`, `city`, `state`, `zip_code`"

// Handler serves GET /api/mass-spending.
type Handler struct {
	client *sc.Client
}

// NewHandler creates a spending handler with the given Socrata app token.
func NewHandler(appToken string, devMode bool) *Handler {
	return &Handler{client: sc.NewClient(appToken, devMode)}
}

// socrataRaw is the raw record shape returned by Socrata query.json.
type socrataRaw struct {
	Vendor             string `json:"vendor"`
	VendorID           string `json:"vendor_id"`
	Department         string `json:"department"`
	DepartmentCode     string `json:"department_code"`
	CabinetSecretariat string `json:"cabinet_secretariat"`
	Amount             string `json:"amount"`
	PaymentID          string `json:"payment_id"`
	PaymentMethod      string `json:"payment_method"`
	Date               string `json:"date"`
	CreateDate         string `json:"create_date"`
	FiscalYear         string `json:"budget_fiscal_year"`
	FiscalPeriod       string `json:"fiscal_period"`
	AppropriationType  string `json:"appropriation_type"`
	AppropriationName  string `json:"appropriation_name"`
	AppropriationCode  string `json:"appropriation_code"`
	ObjectClass        string `json:"object_class"`
	ObjectCode         string `json:"object_code"`
	Fund               string `json:"fund"`
	FundCode           string `json:"fund_code"`
	City               string `json:"city"`
	State              string `json:"state"`
	ZipCode            string `json:"zip_code"`
}

// GetSpendingData handles GET /api/mass-spending.
// Supported query params: year, page, limit, sort, sort_field,
// vendor, org, cabinet, object_class, city, state, fund, from, to.
func (h *Handler) GetSpendingData(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	// TODO: more efficient way of getting params
	// we could probably skip a lot of the variable assigning below if the
	// query is empty

	year := q.Get("year")
	if year == "" {
		year = "2026"
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

	vendor := sc.SanitizeSQL(q.Get("vendor"))
	org := sc.SanitizeSQL(q.Get("org"))
	cabinet := sc.SanitizeSQL(q.Get("cabinet"))
	objectClass := sc.SanitizeSQL(q.Get("object_class"))
	city := sc.SanitizeSQL(q.Get("city"))
	state := sc.SanitizeSQL(q.Get("state"))
	fund := sc.SanitizeSQL(q.Get("fund"))
	fromDate := sc.ValidDate(q.Get("from"))
	toDate := sc.ValidDate(q.Get("to"))

	where := buildWhere(sc.SanitizeSQL(year), vendor, org, cabinet, objectClass, city, state, fund, fromDate, toDate)

	dataSQL := fmt.Sprintf(
		"SELECT %s %s ORDER BY `%s` %s LIMIT %d OFFSET %d",
		selectCols, where, sortCol, sortDir, limit, page*limit,
	)
	aggSQL := fmt.Sprintf("SELECT COUNT(*) as total_count, SUM(`amount`) as total_amount %s", where)

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
		log.Printf("[ERROR] spending data query: %v", dataErr)
		sc.WriteError(w, "failed to fetch spending data", http.StatusBadGateway)
		return
	}
	if aggErr != nil {
		log.Printf("[ERROR] spending aggregate query: %v", aggErr)
		sc.WriteError(w, "failed to fetch spending aggregates", http.StatusBadGateway)
		return
	}

	var raws []socrataRaw
	if err := json.Unmarshal(rawData, &raws); err != nil {
		log.Printf("[ERROR] spending parse records: %v", err)
		sc.WriteError(w, "failed to parse spending data", http.StatusInternalServerError)
		return
	}

	records := make([]models.SpendingRecord, 0, len(raws))
	for _, rec := range raws {
		amount, _ := strconv.ParseFloat(rec.Amount, 64)
		records = append(records, models.SpendingRecord{
			Vendor:             rec.Vendor,
			VendorID:           rec.VendorID,
			Department:         rec.Department,
			DepartmentCode:     rec.DepartmentCode,
			CabinetSecretariat: rec.CabinetSecretariat,
			Amount:             amount,
			PaymentID:          rec.PaymentID,
			PaymentMethod:      rec.PaymentMethod,
			Date:               rec.Date,
			CreateDate:         rec.CreateDate,
			FiscalYear:         rec.FiscalYear,
			FiscalPeriod:       rec.FiscalPeriod,
			AppropriationType:  rec.AppropriationType,
			AppropriationName:  rec.AppropriationName,
			AppropriationCode:  rec.AppropriationCode,
			ObjectClass:        rec.ObjectClass,
			ObjectCode:         rec.ObjectCode,
			Fund:               rec.Fund,
			FundCode:           rec.FundCode,
			City:               rec.City,
			State:              rec.State,
			ZipCode:            rec.ZipCode,
		})
	}

	var aggs []sc.Aggregate
	if err := json.Unmarshal(rawAgg, &aggs); err != nil {
		log.Printf("[ERROR] spending parse aggregates: %v", err)
		sc.WriteError(w, "failed to parse spending aggregates", http.StatusInternalServerError)
		return
	}

	var count int
	var totalAmount float64
	if len(aggs) > 0 {
		count = int(sc.ParseNumber(aggs[0].Count))
		totalAmount = sc.ParseNumber(aggs[0].TotalAmount)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.SpendingResponse{
		Data:        records,
		Count:       count,
		TotalAmount: totalAmount,
	})
}

func buildWhere(year, vendor, org, cabinet, objectClass, city, state, fund, fromDate, toDate string) string {
	conds := []string{fmt.Sprintf("`budget_fiscal_year` = '%s'", year)}
	if vendor != "" {
		conds = append(conds, fmt.Sprintf("UPPER(`vendor`) LIKE UPPER('%%%s%%')", vendor))
	}
	if org != "" {
		conds = append(conds, fmt.Sprintf("UPPER(`department`) LIKE UPPER('%%%s%%')", org))
	}
	if cabinet != "" {
		conds = append(conds, fmt.Sprintf("UPPER(`cabinet_secretariat`) LIKE UPPER('%%%s%%')", cabinet))
	}
	if objectClass != "" {
		conds = append(conds, fmt.Sprintf("UPPER(`object_class`) LIKE UPPER('%%%s%%')", objectClass))
	}
	if city != "" {
		conds = append(conds, fmt.Sprintf("UPPER(`city`) LIKE UPPER('%%%s%%')", city))
	}
	if state != "" {
		conds = append(conds, fmt.Sprintf("UPPER(`state`) LIKE UPPER('%%%s%%')", state))
	}
	if fund != "" {
		conds = append(conds, fmt.Sprintf("UPPER(`fund`) LIKE UPPER('%%%s%%')", fund))
	}
	if fromDate != "" {
		conds = append(conds, fmt.Sprintf("`date` >= '%sT00:00:00.000'", fromDate))
	}
	if toDate != "" {
		conds = append(conds, fmt.Sprintf("`date` <= '%sT23:59:59.999'", toDate))
	}
	return "WHERE " + strings.Join(conds, " AND ")
}

// Sort by date by default.
// TODO: review aliased department fields and their use to see if this mapping
// is necessary.
func mapSortField(f string) string {
	switch f {
	case "vendor":
		return "vendor"
	case "vendor_id":
		return "vendor_id"
	case "amount":
		return "amount"
	case "org2", "department":
		return "department"
	case "org4", "department_code":
		return "department_code"
	case "cabinet_secretariat":
		return "cabinet_secretariat"
	case "object_class":
		return "object_class"
	case "object_code":
		return "object_code"
	case "fund":
		return "fund"
	case "fund_code":
		return "fund_code"
	case "city":
		return "city"
	case "state":
		return "state"
	case "zip_code":
		return "zip_code"
	case "payment_method":
		return "payment_method"
	case "date":
		return "date"
	case "create_date":
		return "create_date"
	default:
		return "date"
	}
}
