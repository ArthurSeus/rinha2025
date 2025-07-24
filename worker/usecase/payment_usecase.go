package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"payment-worker/model"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

type WorkerJob struct {
	Msg         *nats.Msg
	IsBenchmark bool
	Target      string
}

type processorStatus struct {
	Failing    bool
	RespTime   time.Duration
	LastError  error
	LastUpdate time.Time
}

type PaymentUsecase struct {
	natsJS nats.JetStreamContext

	mu             sync.Mutex
	defaultStatus  processorStatus
	fallbackStatus processorStatus
	processor      string
}

func NewPaymentUsecase(natsJS nats.JetStreamContext) *PaymentUsecase {
	return &PaymentUsecase{
		natsJS:    natsJS,
		processor: "default",
	}
}

func (p *PaymentUsecase) StartWorkerPool(ctx context.Context, numWorkers int) {
	// // Deleta consumer antigo se existir (ok para dev/homolog)
	// _ = p.natsJS.DeleteConsumer("PAYMENTS_STREAM", consumerName)

	// // Cria o consumer explicitamente
	// _, err := p.natsJS.AddConsumer("PAYMENTS_STREAM", &nats.ConsumerConfig{
	// 	Durable:       consumerName,
	// 	AckPolicy:     nats.AckExplicitPolicy,
	// 	FilterSubject: "payments_to_send",
	// })
	// if err != nil && !strings.Contains(err.Error(), "consumer already exists") {
	// 	panic(fmt.Sprintf("Erro ao criar AddConsumer: %v", err))
	// }

	// Agora faz bind (não tenta criar de novo)
	sub, err := p.natsJS.PullSubscribe("jobs", "worker-1", nats.BindStream("WORKERS_STREAM"))
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("foi")

	// for {
	// 	// Puxa até 1 mensagem, espera até 1s se não chegar nada
	// 	msgs, err := sub.Fetch(1, nats.MaxWait(1*time.Second))
	// 	if err != nil && err != nats.ErrTimeout {
	// 		log.Println("Erro:", err)
	// 		continue
	// 	}
	// 	for _, msg := range msgs {
	// 		log.Println("Recebido:", string(msg.Data))
	// 		msg.Ack() // Confirma o processamento
	// 	}
	// }

	jobs := make(chan WorkerJob, 1000)

	for i := 0; i < numWorkers; i++ {
		go func(workerId int) {
			for job := range jobs {
				var payment model.PaymentRequestTimed
				if err := json.Unmarshal(job.Msg.Data, &payment); err != nil {
					job.Msg.Ack()
					continue
				}

				if job.IsBenchmark {
					start := time.Now()
					err := p.tryProcessor(job.Target, payment, job.Msg.Data)
					duration := time.Since(start)

					p.mu.Lock()
					status := processorStatus{
						Failing:    err != nil,
						RespTime:   duration,
						LastError:  err,
						LastUpdate: time.Now(),
					}
					if job.Target == "default" {
						p.defaultStatus = status
					} else {
						p.fallbackStatus = status
					}
					p.processor = p.calculateProcessor(p.defaultStatus, p.fallbackStatus)
					p.mu.Unlock()

					job.Msg.Ack()
				} else {
					err := p.sendToProcessor(payment, job.Msg.Data)
					if err != nil {
						_ = p.publish(job.Msg.Data)
					}
					job.Msg.Ack()
				}
			}
		}(i)
	}

	for {
		msgs, err := sub.Fetch(numWorkers, nats.MaxWait(1*time.Second))
		if err != nil {
			continue
		}
		for i, msg := range msgs {
			log.Printf("[DISPATCHER] Nova mensagem lida da fila: %s", string(msg.Data))
			if i < 2 {
				target := "default"
				if i%2 == 1 {
					target = "fallback"
				}
				jobs <- WorkerJob{Msg: msg, IsBenchmark: true, Target: target}
			} else {
				jobs <- WorkerJob{Msg: msg, IsBenchmark: false}
			}
		}
	}
}

func (p *PaymentUsecase) publish(body []byte) error {
	_, err := p.natsJS.Publish("payments_to_send", body)
	return err
}

func (p *PaymentUsecase) sendToProcessor(payment model.PaymentRequestTimed, data []byte) error {
	defaultStatus := p.defaultStatus
	fallbackStatus := p.fallbackStatus
	selected := p.processor

	if selected == "" {
		return fmt.Errorf("both processors unavailable")
	}

	if err := p.tryProcessor(selected, payment, data); err == nil {
		return nil
	}

	other := "default"
	if selected == "default" {
		other = "fallback"
	}

	shouldTryOther := (other == "default" && !defaultStatus.Failing) || (other == "fallback" && !fallbackStatus.Failing)

	if shouldTryOther {
		if err := p.tryProcessor(other, payment, data); err == nil {
			return nil
		}
	}

	return fmt.Errorf("both processors failed")
}

func (p *PaymentUsecase) tryProcessor(processor string, payment model.PaymentRequestTimed, data []byte) error {
	url := fmt.Sprintf("http://payment-processor-%s:8080/payments", processor)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("bad status: %d", resp.StatusCode)
	}

	newPayment := model.Payment{
		CorrelationID: payment.CorrelationID,
		Processor:     processor,
		Amount:        payment.Amount,
		RequestedAt:   payment.RequestedAt,
	}
	ctx := context.Background()
	if err := p.savePayment(ctx, newPayment); err != nil {
		return fmt.Errorf("failed to persist payment in NATS: %w", err)
	}

	return nil
}

func (p *PaymentUsecase) calculateProcessor(defaultStatus, fallbackStatus processorStatus) string {
	switch {
	case defaultStatus.Failing && fallbackStatus.Failing:
		return ""
	case defaultStatus.Failing:
		return "fallback"
	case fallbackStatus.Failing || defaultStatus.RespTime == fallbackStatus.RespTime ||
		(defaultStatus.RespTime <= 100*time.Millisecond && fallbackStatus.RespTime < defaultStatus.RespTime/3):
		return "default"
	default:
		return "default"
	}
}

func (p *PaymentUsecase) savePayment(ctx context.Context, payment model.Payment) error {
	data, err := json.Marshal(payment)
	if err != nil || data == nil {
		return fmt.Errorf("failed to marshal payment: %w", err)
	}
	_, err = p.natsJS.Publish("persist", data)
	return err
}
