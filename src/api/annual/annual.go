// Package annual serves cross-year aggregate comparisons of spending and revenue.
package annual

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"

	sc "massdata/api/socrata"
)

const (
	spendingViewID = "pegc-naaa"
	revenueViewID  = "kcy7-ivxi"
)

// DeptTotal is a single department/category row returned inside a YearSummary.
type DeptTotal struct {
	Dept  string  `json:"dept"`
	Total float64 `json:"total"`
}

// YearSummary is the per-year aggregate for one dataset.
type YearSummary struct {
	Year   string      `json:"year"`
	Total  float64     `json:"total"`
	Count  int64       `json:"count"`
	ByDept []DeptTotal `json:"by_dept"`
}

// AnnualResponse is the JSON envelope returned by /api/mass-annual.
type AnnualResponse struct {
	Spending []YearSummary `json:"spending"`
	Revenue  []YearSummary `json:"revenue"`
}

// Handler serves GET /api/mass-annual.
type Handler struct {
	client *sc.Client
}

// NewHandler creates an annual handler with the given Socrata app token.
func NewHandler(appToken string, devMode bool) *Handler {
	return &Handler{client: sc.NewClient(appToken, devMode)}
}

// GetAnnualData handles GET /api/mass-annual.
// Query params:
//
//	spending_years=2022,2023,2024   – comma-separated fiscal years for spending
//	revenue_years=2022,2023,2024    – comma-separated fiscal years for revenue
func (h *Handler) GetAnnualData(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	spendingYears := parseYears(q.Get("spending_years"))
	revenueYears  := parseYears(q.Get("revenue_years"))

	var (
		spending    []YearSummary
		revenue     []YearSummary
		spendingErr error
		revenueErr  error
		wg          sync.WaitGroup
	)

	if len(spendingYears) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			spending, spendingErr = h.fetchSpending(spendingYears)
		}()
	}
	if len(revenueYears) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			revenue, revenueErr = h.fetchRevenue(revenueYears)
		}()
	}
	wg.Wait()

	if spendingErr != nil {
		log.Printf("[ERROR] annual spending: %v", spendingErr)
		sc.WriteError(w, "failed to fetch annual spending data", http.StatusBadGateway)
		return
	}
	if revenueErr != nil {
		log.Printf("[ERROR] annual revenue: %v", revenueErr)
		sc.WriteError(w, "failed to fetch annual revenue data", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AnnualResponse{Spending: spending, Revenue: revenue})
}

// fetchSpending runs totals + top-10 department queries concurrently for each
// requested year, then assembles a YearSummary slice in order.
func (h *Handler) fetchSpending(years []string) ([]YearSummary, error) {
	type rawDept struct {
		Department  string `json:"department"`
		TotalAmount string `json:"total_amount"`
	}

	results := make([]YearSummary, len(years))
	errs    := make([]error, len(years))
	var wg sync.WaitGroup

	for i, yr := range years {
		wg.Add(1)
		go func(idx int, year string) {
			defer wg.Done()
			safe  := sc.SanitizeSQL(year)
			where := fmt.Sprintf("WHERE `budget_fiscal_year` = '%s'", safe)

			totSQL  := fmt.Sprintf("SELECT COUNT(*) as total_count, SUM(`amount`) as total_amount %s", where)
			deptSQL := fmt.Sprintf(
				"SELECT `department`, SUM(`amount`) as total_amount %s GROUP BY `department` ORDER BY SUM(`amount`) DESC LIMIT 10",
				where,
			)

			var rawTot, rawDepts json.RawMessage
			var totErr, deptErr error
			var inner sync.WaitGroup
			inner.Add(2)
			go func() { defer inner.Done(); rawTot, totErr = h.client.Query(spendingViewID, totSQL) }()
			go func() { defer inner.Done(); rawDepts, deptErr = h.client.Query(spendingViewID, deptSQL) }()
			inner.Wait()
			if totErr != nil { errs[idx] = totErr; return }
			if deptErr != nil { errs[idx] = deptErr; return }

			var agg []sc.Aggregate
			if err := json.Unmarshal(rawTot, &agg); err != nil { errs[idx] = err; return }
			var depts []rawDept
			if err := json.Unmarshal(rawDepts, &depts); err != nil { errs[idx] = err; return }

			var total float64
			var count int64
			if len(agg) > 0 {
				total = sc.ParseNumber(agg[0].TotalAmount)
				count = int64(sc.ParseNumber(agg[0].Count))
			}
			byDept := make([]DeptTotal, 0, len(depts))
			for _, d := range depts {
				if d.Department == "" { continue }
				t, _ := strconv.ParseFloat(d.TotalAmount, 64)
				byDept = append(byDept, DeptTotal{Dept: d.Department, Total: t})
			}
			results[idx] = YearSummary{Year: year, Total: total, Count: count, ByDept: byDept}
		}(i, yr)
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil { return nil, err }
	}
	return results, nil
}

// fetchRevenue runs totals + top-10 department queries for each requested year.
func (h *Handler) fetchRevenue(years []string) ([]YearSummary, error) {
	type rawDept struct {
		DepartmentName string `json:"department_name"`
		TotalAmount    string `json:"total_amount"`
	}

	results := make([]YearSummary, len(years))
	errs    := make([]error, len(years))
	var wg sync.WaitGroup

	for i, yr := range years {
		wg.Add(1)
		go func(idx int, year string) {
			defer wg.Done()
			safe  := sc.SanitizeSQL(year)
			where := fmt.Sprintf("WHERE `fiscal_year` = '%s'", safe)

			totSQL  := fmt.Sprintf("SELECT COUNT(*) as total_count, SUM(`revenue_collected`) as total_amount %s", where)
			deptSQL := fmt.Sprintf(
				"SELECT `department_name`, SUM(`revenue_collected`) as total_amount %s GROUP BY `department_name` ORDER BY SUM(`revenue_collected`) DESC LIMIT 10",
				where,
			)

			var rawTot, rawDepts json.RawMessage
			var totErr, deptErr error
			var inner sync.WaitGroup
			inner.Add(2)
			go func() { defer inner.Done(); rawTot, totErr = h.client.Query(revenueViewID, totSQL) }()
			go func() { defer inner.Done(); rawDepts, deptErr = h.client.Query(revenueViewID, deptSQL) }()
			inner.Wait()
			if totErr != nil { errs[idx] = totErr; return }
			if deptErr != nil { errs[idx] = deptErr; return }

			var agg []sc.Aggregate
			if err := json.Unmarshal(rawTot, &agg); err != nil { errs[idx] = err; return }
			var depts []rawDept
			if err := json.Unmarshal(rawDepts, &depts); err != nil { errs[idx] = err; return }

			var total float64
			var count int64
			if len(agg) > 0 {
				total = sc.ParseNumber(agg[0].TotalAmount)
				count = int64(sc.ParseNumber(agg[0].Count))
			}
			byDept := make([]DeptTotal, 0, len(depts))
			for _, d := range depts {
				if d.DepartmentName == "" { continue }
				t, _ := strconv.ParseFloat(d.TotalAmount, 64)
				byDept = append(byDept, DeptTotal{Dept: d.DepartmentName, Total: t})
			}
			results[idx] = YearSummary{Year: year, Total: total, Count: count, ByDept: byDept}
		}(i, yr)
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil { return nil, err }
	}
	return results, nil
}

// parseYears splits a comma-separated string into validated 4-digit year strings.
// Accepts up to 8 years maximum to limit concurrent Socrata requests.
func parseYears(s string) []string {
	if s == "" { return nil }
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if len(p) == 4 && p >= "2000" && p <= "2099" {
			out = append(out, p)
		}
	}
	if len(out) > 8 { out = out[:8] }
	return out
}
