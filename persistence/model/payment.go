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
