package main

import (
	"context"
	"os"
	"payment-worker/config"
	"payment-worker/usecase"
	"strconv"
)

func main() {
	nats := config.JetStreamInit()

	paymentUsecase := usecase.NewPaymentUsecase(nats)

	numWorkers, _ := strconv.Atoi(os.Getenv("NUM_WORKERS"))
	paymentUsecase.StartWorkerPool(context.Background(), numWorkers)

	select {}
}
