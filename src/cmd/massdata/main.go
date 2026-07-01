package main

import (
	"log"
	"net/http"
	"os"

	"massdata/api/annual"
	"massdata/api/payroll"
	"massdata/api/revenue"
	"massdata/api/settlements"
	"massdata/api/spending"
	"massdata/config"
)

type Server struct {
	cfg *config.Config
	mux *http.ServeMux
}

func New(cfg *config.Config) *Server {
	s := &Server{
		cfg: cfg,
		mux: http.NewServeMux(),
	}

	devMode := cfg.DevMode == "true"

	s.mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	spendingHandler := spending.NewHandler(cfg.SocrataAppToken, devMode)
	s.mux.HandleFunc("/api/mass-spending", spendingHandler.GetSpendingData)

	revenueHandler := revenue.NewHandler(cfg.SocrataAppToken, devMode)
	s.mux.HandleFunc("/api/mass-revenue", revenueHandler.GetRevenueData)

	payrollHandler := payroll.NewHandler(cfg.SocrataAppToken, devMode)
	s.mux.HandleFunc("/api/mass-payroll", payrollHandler.GetPayrollData)

	annualHandler := annual.NewHandler(cfg.SocrataAppToken, devMode)
	s.mux.HandleFunc("/api/mass-annual", annualHandler.GetAnnualData)

	settlementsHandler := settlements.NewHandler(cfg.SocrataAppToken, devMode)
	s.mux.HandleFunc("/api/mass-settlements", settlementsHandler.GetSettlementsData)

	// Only serve prebuilt frontend assets when running locally in dev mode.
	// In production, the GCP load balancer routes non-/api/* requests to Cloud
	// Storage where the ui/dist directory is served.
	if devMode {
		distDir := "../../ui/dist"
		log.Printf("Serving static files from %s", distDir)
		fs := http.FileServer(http.Dir(distDir))
		s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			path := distDir + r.URL.Path
			if _, err := os.Stat(path); os.IsNotExist(err) {
				http.ServeFile(w, r, distDir+"/index.html")
				return
			}
			fs.ServeHTTP(w, r)
		})
	}

	return s
}

func (s *Server) Handler() http.Handler {
	return corsMiddleware(s.mux)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	wd, _ := os.Getwd()
	log.Printf("working directory: %s", wd)

	cfg := config.Load()
	srv := New(&cfg)
	log.Printf("massdata API listening on %s", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, srv.Handler()); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
