//

package usecase

import (
	"encoding/json"
	"log"
	"math"
	"payment-persistence/model"
	"payment-persistence/repository"
	"time"

	"github.com/nats-io/nats.go"
)

const (
	SubjectNamePersistence = "PERSISTENCE.payment"
)

type PaymentUsecase struct {
	Repo   *repository.MemoryPaymentRepository
	natsJS nats.JetStreamContext
}

func NewPaymentUsecase(repo *repository.MemoryPaymentRepository, natsJS nats.JetStreamContext) *PaymentUsecase {
	return &PaymentUsecase{
		Repo:   repo,
		natsJS: natsJS,
	}
}

func round2(val float64) float64 {
	return math.Round(val*100) / 100
}

func (p *PaymentUsecase) GetPaymentsSummary(from, to *time.Time) (model.Summary, error) {
	all := p.Repo.GetAll()
	var defaultCount, fallbackCount int64
	var defaultAmount, fallbackAmount float64

	for _, pay := range all {
		if (from == nil || !pay.RequestedAt.Before(*from)) && (to == nil || !pay.RequestedAt.After(*to)) {
			switch pay.Processor {
			case "default":
				defaultCount++
				defaultAmount += pay.Amount.InexactFloat64()
			case "fallback":
				fallbackCount++
				fallbackAmount += pay.Amount.InexactFloat64()
			}
		}
	}

	return model.Summary{
		DefaultProcessor: model.PaymentSummary{
			TotalRequests: defaultCount,
			TotalAmount:   round2(defaultAmount),
		},
		FallbackProcessor: model.PaymentSummary{
			TotalRequests: fallbackCount,
			TotalAmount:   round2(fallbackAmount),
		},
	}, nil
}

func (p *PaymentUsecase) PurgeAll() error {
	p.Repo.Purge()

	return nil
}

func (p *PaymentUsecase) StartWorkerPool(numWorkers int) {
	jobs := make(chan *nats.Msg, 1000)

	for workerId := 0; workerId < numWorkers; workerId++ {
		go func(workerId int) {
			for msg := range jobs {
				var payment model.Payment
				err := json.Unmarshal(msg.Data, &payment)
				if err != nil {
					log.Printf("[worker %d] erro no Unmarshal: %v", workerId, err)
					_ = msg.Ack()
					continue
				}
				_ = p.Repo.Save(&payment)
				_ = msg.Ack()
			}
		}(workerId)
	}

	_, err := p.natsJS.Subscribe(SubjectNamePersistence, func(m *nats.Msg) {
		jobs <- m
	}, nats.Durable("payments-worker"), nats.ManualAck())

	if err != nil {
		log.Println("Subscribe failed:", err)
	}
}
