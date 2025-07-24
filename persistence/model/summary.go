package model

type Summary struct {
	DefaultProcessor  PaymentSummary `json:"default"`
	FallbackProcessor PaymentSummary `json:"fallback"`
}

type PaymentSummary struct {
	TotalRequests int64   `json:"totalRequests"`
	TotalAmount   float64 `json:"totalAmount"`
}
