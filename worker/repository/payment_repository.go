package repository

import (
	"payment-worker/model"
)

type PaymentRepository interface {
	Save(payment *model.Payment) error
}
