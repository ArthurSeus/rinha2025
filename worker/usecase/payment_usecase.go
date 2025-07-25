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

const (
	SubjectNamePayment     = "PAYMENTS.payment"
	SubjectNamePersistence = "PERSISTENCE.payment"
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
	jobs := make(chan *nats.Msg, 1000)

	for workerId := 0; workerId < numWorkers; workerId++ {
		go func(workerId int) {
			for msg := range jobs {
				var payment model.PaymentRequestTimed
				err := json.Unmarshal(msg.Data, &payment)
				if err != nil {
					log.Printf("[worker %d] erro no Unmarshal: %v", workerId, err)
					_ = p.publishNATS(msg.Data)
					_ = msg.Ack()
					continue
				}

				if workerId < 20 {
					target := "default"
					if workerId%2 != 0 {
						target = "fallback"
					}
					err := p.benchmarkProcessor(target, payment, msg.Data)
					if err != nil {
						_ = p.publishNATS(msg.Data)
					}
				} else {
					err := p.sendToProcessor(payment, msg.Data)
					if err != nil {
						_ = p.publishNATS(msg.Data)
					}
				}

				_ = msg.Ack()
			}
		}(workerId)
	}

	_, err := p.natsJS.Subscribe(SubjectNamePayment, func(m *nats.Msg) {
		jobs <- m
	}, nats.Durable("payments-worker"), nats.ManualAck())

	if err != nil {
		log.Println("Subscribe failed:", err)
	}
}

func (p *PaymentUsecase) benchmarkProcessor(target string, payment model.PaymentRequestTimed, data []byte) error {
	start := time.Now()
	err := p.tryProcessor(target, payment, data)
	duration := time.Since(start)

	p.mu.Lock()
	defer p.mu.Unlock()

	status := processorStatus{
		Failing:    err != nil,
		RespTime:   duration,
		LastError:  err,
		LastUpdate: time.Now(),
	}
	if target == "default" {
		p.defaultStatus = status
	} else {
		p.fallbackStatus = status
	}
	p.processor = p.calculateProcessor(p.defaultStatus, p.fallbackStatus)

	if err != nil {
		return err
	}

	return nil
}

func (p *PaymentUsecase) publishNATS(body []byte) error {
	_, err := p.natsJS.Publish(SubjectNamePayment, body)
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
	if err := p.savePayment(newPayment); err != nil {
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

func (p *PaymentUsecase) savePayment(payment model.Payment) error {
	body, err := json.Marshal(payment)
	if err != nil || body == nil {
		return fmt.Errorf("failed to marshal payment: %w", err)
	}
	_, err = p.natsJS.Publish(SubjectNamePersistence, body)
	return err
}
