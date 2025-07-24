package main

import (
	"context"
	"os"
	"payment-worker/config"
	"payment-worker/usecase"
	"strconv"
)

func main() {
	config.InitNATS()

	paymentUsecase := usecase.NewPaymentUsecase(config.NatsJS)

	numWorkers, _ := strconv.Atoi(os.Getenv("NUM_WORKERS"))
	paymentUsecase.StartWorkerPool(context.Background(), numWorkers)

	select {}
}
