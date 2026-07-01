// Package socrata provides a shared HTTP client for the Massachusetts
// Socrata open data API (cthru.data.socrata.com).
package socrata

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const baseURL = "https://cthru.data.socrata.com/api/v3/views"

// Client handles authenticated requests to the Socrata query.json API.
type Client struct {
	AppToken string
	DevMode  bool
	http     *http.Client
}

// NewClient returns a Client configured with the given app token.
// We have an extended timeout here since some new queries take
// several seconds to get a response from CTHRU.
func NewClient(appToken string, devMode bool) *Client {
	return &Client{
		AppToken: appToken,
		DevMode:  devMode,
		http: &http.Client{
			Timeout: 300 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 20,
				IdleConnTimeout:     600 * time.Second,
			},
		},
	}
}

// Query runs a SQL SELECT against the Socrata query.json endpoint for given
// viewID.
func (c *Client) Query(viewID, sql string) (json.RawMessage, error) {
	url := fmt.Sprintf("%s/%s/query.json", baseURL, viewID)
	body, err := json.Marshal(map[string]string{"query": sql})
	if err != nil {
		return nil, err
	}
	if c.DevMode {
		log.Printf("[DEBUG] Socrata POST %s body=%s", url, body)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-App-Token", c.AppToken)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("upstream %d: %.200s", resp.StatusCode, data)
	}
	if !json.Valid(data) {
		return nil, fmt.Errorf("invalid JSON from upstream: %.200s", data)
	}
	return json.RawMessage(data), nil
}

// Aggregate holds COUNT/SUM results from a Socrata aggregate query.
type Aggregate struct {
	Count       json.RawMessage `json:"total_count"`
	TotalAmount json.RawMessage `json:"total_amount"`
}

// ParseNumber handles Socrata returning COUNT/SUM as either a JSON number or string.
func ParseNumber(raw json.RawMessage) float64 {
	if len(raw) == 0 {
		return 0
	}
	var f float64
	if err := json.Unmarshal(raw, &f); err == nil {
		return f
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		f, _ := strconv.ParseFloat(s, 64)
		return f
	}
	return 0
}

// SanitizeSQL escapes single quotes to prevent SQL injection.
func SanitizeSQL(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// ValidDate returns s if it is a valid YYYY-MM-DD date, otherwise "".
func ValidDate(s string) string {
	if _, err := time.Parse("2006-01-02", s); err != nil {
		return ""
	}
	return s
}

// WriteError writes a JSON error response.
func WriteError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	fmt.Fprintf(w, `{"error":%q}`, msg)
}
