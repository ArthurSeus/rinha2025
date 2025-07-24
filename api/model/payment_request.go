package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type PaymentRequest struct {
	CorrelationId uuid.UUID `json:"correlationId"`
	Amount        float64   `json:"amount"`
}

type PaymentRequestTimed struct {
	CorrelationID uuid.UUID       `json:"correlationId"`
	Amount        decimal.Decimal `json:"amount"`
	RequestedAt   time.Time       `json:"requestedAt"`
}
