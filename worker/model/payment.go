package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Payment struct {
	CorrelationID uuid.UUID
	Processor     string
	Amount        decimal.Decimal
	RequestedAt   time.Time
}

type PaymentRequestTimed struct {
	CorrelationID uuid.UUID       `json:"correlationId"`
	Amount        decimal.Decimal `json:"amount"`
	RequestedAt   time.Time       `json:"requestedAt"`
}
