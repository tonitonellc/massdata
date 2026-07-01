package models

// SpendingRecord represents a single state spending transaction from the Socrata API.
type SpendingRecord struct {
	Vendor             string  `json:"vendor"`
	VendorID           string  `json:"vendor_id"`
	Department         string  `json:"department"`
	DepartmentCode     string  `json:"department_code"`
	CabinetSecretariat string  `json:"cabinet_secretariat"`
	Amount             float64 `json:"amount"`
	PaymentID          string  `json:"payment_id"`
	PaymentMethod      string  `json:"payment_method"`
	Date               string  `json:"date"`
	CreateDate         string  `json:"create_date"`
	FiscalYear         string  `json:"budget_fiscal_year"`
	FiscalPeriod       string  `json:"fiscal_period"`
	AppropriationType  string  `json:"appropriation_type"`
	AppropriationName  string  `json:"appropriation_name"`
	AppropriationCode  string  `json:"appropriation_code"`
	ObjectClass        string  `json:"object_class"`
	ObjectCode         string  `json:"object_code"`
	Fund               string  `json:"fund"`
	FundCode           string  `json:"fund_code"`
	City               string  `json:"city"`
	State              string  `json:"state"`
	ZipCode            string  `json:"zip_code"`
}

// SpendingResponse is the top-level envelope returned to the frontend.
type SpendingResponse struct {
	Data        []SpendingRecord `json:"data"`
	Count       int              `json:"count"`
	TotalAmount float64          `json:"total_amount"`
}

// RevenueRecord represents a single revenue transaction from the Socrata API.
type RevenueRecord struct {
	FiscalYear          string  `json:"fiscal_year"`
	FiscalPeriod        string  `json:"fiscal_period"`
	FiscalPeriodMonth   string  `json:"fiscal_period_month"`
	ApdNm               string  `json:"apd_nm"`
	RevenueCategoryName string  `json:"revenue_category_name"`
	FundCategoryName    string  `json:"fund_category_name"`
	FundNumber          string  `json:"fund_number"`
	FundName            string  `json:"fund_name"`
	SubFund             string  `json:"sub_fund"`
	SubFundName         string  `json:"sub_fund_name"`
	RevenueClassName    string  `json:"revenue_class_name"`
	CabinetName         string  `json:"cabinet_name"`
	DepartmentCode      string  `json:"department_code"`
	DepartmentName      string  `json:"department_name"`
	RevenueSource       string  `json:"revenue_source"`
	RevenueSourceName   string  `json:"revenue_source_name"`
	RevenueCollected    float64 `json:"revenue_collected"`
	TransNo             string  `json:"trans_no"`
}

// RevenueResponse is the top-level envelope returned to the frontend.
type RevenueResponse struct {
	Data        []RevenueRecord `json:"data"`
	Count       int             `json:"count"`
	TotalAmount float64         `json:"total_amount"`
}

// PayrollRecord represents a single payroll record from the Socrata API.
type PayrollRecord struct {
	ID                   string  `json:"id"`
	Year                 string  `json:"year"`
	TransNo              string  `json:"trans_no"`
	NameLast             string  `json:"name_last"`
	NameFirst            string  `json:"name_first"`
	DepartmentDivision   string  `json:"department_division"`
	DepartmentCode       string  `json:"department_code"`
	PositionTitle        string  `json:"position_title"`
	PositionType         string  `json:"position_type"`
	ServiceEndDate       string  `json:"service_end_date"`
	PayTotalActual       float64 `json:"pay_total_actual"`
	PayBaseActual        float64 `json:"pay_base_actual"`
	PayBuyoutActual      float64 `json:"pay_buyout_actual"`
	PayOvertimeActual    float64 `json:"pay_overtime_actual"`
	PayOtherActual       float64 `json:"pay_other_actual"`
	AnnualRate           float64 `json:"annual_rate"`
	PayYearToDate        float64 `json:"pay_year_to_date"`
	DeptLocationZipCode  string  `json:"department_location_zip_code"`
	Contract             string  `json:"contract"`
	BargainingGroupNo    string  `json:"bargaining_group_no"`
	BargainingGroupTitle string  `json:"bargaining_group_title"`
}

// PayrollResponse is the top-level envelope returned to the frontend.
type PayrollResponse struct {
	Data        []PayrollRecord `json:"data"`
	Count       int             `json:"count"`
	TotalAmount float64         `json:"total_amount"`
}

// SettlementsRecord represents a single settlement or judgment payment from the Socrata API.
type SettlementsRecord struct {
	PaidOnBehalfOf     string  `json:"paid_on_behalf_of"`
	PaymentDate        string  `json:"payment_date"`
	LineAmount         float64 `json:"line_amount"`
	PayeeName          string  `json:"payee_name"`
	BFY                string  `json:"bfy"`
	Quarter            string  `json:"quarter"`
	DeptPaidOnBehalfOf string  `json:"dept_paid_on_behalf_of"`
}

// SettlementsResponse is the top-level envelope returned to the frontend.
type SettlementsResponse struct {
	Data        []SettlementsRecord `json:"data"`
	Count       int                 `json:"count"`
	TotalAmount float64             `json:"total_amount"`
}
