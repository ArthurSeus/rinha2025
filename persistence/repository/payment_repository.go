package repository

import (
	"payment-persistence/model"
)

type PaymentRepository interface {
	Save(payment *model.Payment) error
}
