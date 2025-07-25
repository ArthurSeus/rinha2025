package usecase

import (
	"api/model"
	"encoding/json"
	"log"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/shopspring/decimal"
)

const (
	SubjectNamePayment = "PAYMENTS.payment"
)

type PaymentUsecase struct {
	natsJS nats.JetStreamContext
}

func NewPaymentUsecase(natsJS nats.JetStreamContext) *PaymentUsecase {
	return &PaymentUsecase{
		natsJS: natsJS,
	}
}

func (p *PaymentUsecase) HandlePaymentRequest(body []byte) {
	var req model.PaymentRequest
	if err := json.Unmarshal(body, &req); err != nil {
		log.Printf("Erro ao decodificar: %v", err)
		return
	}
	_ = p.PostPayment(req)
}

func (p *PaymentUsecase) PostPayment(payment model.PaymentRequest) error {
	amount := decimal.NewFromFloat(payment.Amount)
	paymentTimed := model.PaymentRequestTimed{
		CorrelationID: payment.CorrelationId,
		Amount:        amount,
		RequestedAt:   time.Now().UTC(),
	}

	body, err := json.Marshal(paymentTimed)
	if err != nil {
		return err
	}

	err = p.publishNATS(body)
	if err != nil {
		return err
	}

	return nil
}

func (p *PaymentUsecase) publishNATS(body []byte) error {
	_, err := p.natsJS.Publish(SubjectNamePayment, body)
	return err
}
