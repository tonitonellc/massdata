// Package settlements proxies requests to the Massachusetts Socrata settlements API.
package settlements

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

const viewID = "gpqz-7ppn"

const selectCols = "`paid_on_behalf_of`, `payment_date`, `line_amount`, `payee_name`, `bfy`, `quarter`, `dept_paid_on_behalf_of`"

// Handler serves GET /api/mass-settlements.
type Handler struct {
	client *sc.Client
}

// NewHandler creates a settlements handler with the given Socrata app token.
func NewHandler(appToken string, devMode bool) *Handler {
	return &Handler{client: sc.NewClient(appToken, devMode)}
}

type socrataRaw struct {
	PaidOnBehalfOf     string `json:"paid_on_behalf_of"`
	PaymentDate        string `json:"payment_date"`
	LineAmount         string `json:"line_amount"`
	PayeeName          string `json:"payee_name"`
	BFY                string `json:"bfy"`
	Quarter            string `json:"quarter"`
	DeptPaidOnBehalfOf string `json:"dept_paid_on_behalf_of"`
}

// GetSettlementsData handles GET /api/mass-settlements.
// Supported query params: year (bfy), page, limit, sort, sort_field,
// payee, dept, quarter, from, to.
func (h *Handler) GetSettlementsData(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	year := sc.SanitizeSQL(q.Get("year"))
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

	payee := sc.SanitizeSQL(q.Get("payee"))
	dept := sc.SanitizeSQL(q.Get("dept"))
	quarter := sc.SanitizeSQL(q.Get("quarter"))
	fromDate := sc.ValidDate(q.Get("from"))
	toDate := sc.ValidDate(q.Get("to"))

	where := buildWhere(year, payee, dept, quarter, fromDate, toDate)

	dataSQL := fmt.Sprintf(
		"SELECT %s %s ORDER BY `%s` %s LIMIT %d OFFSET %d",
		selectCols, where, sortCol, sortDir, limit, page*limit,
	)
	aggSQL := fmt.Sprintf("SELECT COUNT(*) as total_count, SUM(`line_amount`) as total_amount %s", where)

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
		log.Printf("[ERROR] settlements data query: %v", dataErr)
		sc.WriteError(w, "failed to fetch settlements data", http.StatusBadGateway)
		return
	}
	if aggErr != nil {
		log.Printf("[ERROR] settlements aggregate query: %v", aggErr)
		sc.WriteError(w, "failed to fetch settlements aggregates", http.StatusBadGateway)
		return
	}

	var raws []socrataRaw
	if err := json.Unmarshal(rawData, &raws); err != nil {
		log.Printf("[ERROR] settlements parse records: %v", err)
		sc.WriteError(w, "failed to parse settlements data", http.StatusInternalServerError)
		return
	}

	records := make([]models.SettlementsRecord, 0, len(raws))
	for _, rec := range raws {
		amount, _ := strconv.ParseFloat(rec.LineAmount, 64)
		records = append(records, models.SettlementsRecord{
			PaidOnBehalfOf:     rec.PaidOnBehalfOf,
			PaymentDate:        rec.PaymentDate,
			LineAmount:         amount,
			PayeeName:          rec.PayeeName,
			BFY:                rec.BFY,
			Quarter:            rec.Quarter,
			DeptPaidOnBehalfOf: rec.DeptPaidOnBehalfOf,
		})
	}

	var aggs []sc.Aggregate
	if err := json.Unmarshal(rawAgg, &aggs); err != nil {
		log.Printf("[ERROR] settlements parse aggregates: %v", err)
		sc.WriteError(w, "failed to parse settlements aggregates", http.StatusInternalServerError)
		return
	}

	var count int
	var totalAmount float64
	if len(aggs) > 0 {
		count = int(sc.ParseNumber(aggs[0].Count))
		totalAmount = sc.ParseNumber(aggs[0].TotalAmount)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.SettlementsResponse{
		Data:        records,
		Count:       count,
		TotalAmount: totalAmount,
	})
}

func buildWhere(year, payee, dept, quarter, fromDate, toDate string) string {
	conds := []string{fmt.Sprintf("`bfy` = '%s'", year)}
	if payee != "" {
		conds = append(conds, fmt.Sprintf("UPPER(`payee_name`) LIKE UPPER('%%%s%%')", payee))
	}
	if dept != "" {
		conds = append(conds, fmt.Sprintf("UPPER(`dept_paid_on_behalf_of`) LIKE UPPER('%%%s%%')", dept))
	}
	if quarter != "" {
		conds = append(conds, fmt.Sprintf("`quarter` = '%s'", quarter))
	}
	if fromDate != "" {
		conds = append(conds, fmt.Sprintf("`payment_date` >= '%sT00:00:00.000'", fromDate))
	}
	if toDate != "" {
		conds = append(conds, fmt.Sprintf("`payment_date` <= '%sT23:59:59.999'", toDate))
	}
	return "WHERE " + strings.Join(conds, " AND ")
}

// TODO: review this mapping and remove if not needed, as it doesn't
// appear to be mapping anything.
func mapSortField(f string) string {
	switch f {
	case "payee_name":
		return "payee_name"
	case "line_amount":
		return "line_amount"
	case "bfy":
		return "bfy"
	case "quarter":
		return "quarter"
	case "dept_paid_on_behalf_of":
		return "dept_paid_on_behalf_of"
	case "paid_on_behalf_of":
		return "paid_on_behalf_of"
	default:
		return "payment_date"
	}
}
