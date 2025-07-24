//

package usecase

import (
	"log"
	"math"
	"payment-persistence/model"
	"payment-persistence/repository"
	"time"

	"github.com/nats-io/nats.go"
)

type PaymentUsecase struct {
	Repo *repository.MemoryPaymentRepository
	Nats nats.JetStreamContext
}

func NewPaymentUsecase(repo *repository.MemoryPaymentRepository, natsJS nats.JetStreamContext) *PaymentUsecase {
	return &PaymentUsecase{
		Repo: repo,
		Nats: natsJS,
	}
}

func round2(val float64) float64 {
	return math.Round(val*100) / 100
}

func (u *PaymentUsecase) GetPaymentsSummary(from, to *time.Time) (model.Summary, error) {
	all := u.Repo.GetAll()
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

func (u *PaymentUsecase) PurgeAll() error {
	u.Repo.Purge()

	if err := u.Nats.PurgeStream("PAYMENTS_STREAM"); err != nil && err != nats.ErrStreamNotFound {
		return err
	}
	if err := u.Nats.PurgeStream("PAYMENTS_TO_PERSIST"); err != nil && err != nats.ErrStreamNotFound {
		return err
	}

	return nil
}

func (u *PaymentUsecase) StartPersistenceWorkers(numWorkers int) {
	sub, err := u.Nats.PullSubscribe("persist", "worker-1", nats.BindStream("WORKERS_STREAM"))
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("foi2")

	for {
		// Puxa até 1 mensagem, espera até 1s se não chegar nada
		msgs, err := sub.Fetch(1, nats.MaxWait(1*time.Second))
		if err != nil && err != nats.ErrTimeout {
			log.Println("Erro:", err)
			continue
		}
		for _, msg := range msgs {
			log.Println("Recebido:", string(msg.Data))
			msg.Ack() // Confirma o processamento
		}
	}

	// const consumerName = "payments_to_persist"

	// sub, err := u.Nats.PullSubscribe(
	// 	"PAYMENTS_TO_PERSIST",
	// 	consumerName,
	// 	nats.BindStream("PAYMENTS_TO_PERSIST"),
	// )
	// if err != nil {
	// 	log.Fatalf("Erro ao criar PullSubscribe NATS: %v", err)
	// }

	// jobs := make(chan *nats.Msg, 1000)

	// // Workers (goroutines)
	// for range numWorkers {
	// 	go func() {
	// 		for msg := range jobs {
	// 			var payment model.Payment
	// 			if err := json.Unmarshal(msg.Data, &payment); err != nil {
	// 				msg.Ack()
	// 				continue
	// 			}
	// 			_ = u.Repo.Save(&payment)
	// 			msg.Ack()
	// 		}
	// 	}()
	// }

	// // Loop central faz fetch e despacha para o pool
	// go func() {
	// 	for {
	// 		msgs, err := sub.Fetch(10, nats.MaxWait(1*time.Second))
	// 		print("%s", msgs)
	// 		if err != nil {
	// 			if err == nats.ErrTimeout {
	// 				continue
	// 			}
	// 			time.Sleep(50 * time.Millisecond)
	// 			continue
	// 		}
	// 		for _, msg := range msgs {
	// 			log.Printf("[PERSIST WORKER] Nova mensagem lida da fila: %s", string(msg.Data))
	// 			jobs <- msg
	// 		}
	// 	}
	// }()
}
