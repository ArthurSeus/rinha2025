package repository

import (
	"payment-persistence/model"
	"sync"
)

type MemoryPaymentRepository struct {
	payments sync.Map // key: CorrelationId, value: *model.Payment
}

func NewMemoryPaymentRepository() *MemoryPaymentRepository {
	return &MemoryPaymentRepository{}
}

func (r *MemoryPaymentRepository) Save(payment *model.Payment) error {
	r.payments.Store(payment.CorrelationID, payment)
	return nil
}

func (r *MemoryPaymentRepository) GetAll() []*model.Payment {
	result := make([]*model.Payment, 0)
	r.payments.Range(func(_, value any) bool {
		if p, ok := value.(*model.Payment); ok {
			result = append(result, p)
		}
		return true
	})
	return result
}

func (r *MemoryPaymentRepository) Purge() {
	r.payments = sync.Map{} // simplesmente reseta o mapa
}
