package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"payment-persistence/config"
	"payment-persistence/repository"
	"payment-persistence/usecase"
	"strconv"
	"time"

	"golang.org/x/net/http2"
)

func main() {
	nats := config.JetStreamInit()

	paymentRepository := repository.NewMemoryPaymentRepository()
	paymentUsecase := usecase.NewPaymentUsecase(paymentRepository, nats)

	numWorkers, _ := strconv.Atoi(os.Getenv("NUM_WORKERS"))
	paymentUsecase.StartWorkerPool(numWorkers)

	mux := http.NewServeMux()
	mux.HandleFunc("/payments-summary", PaymentsSummaryHandler(paymentUsecase))
	mux.HandleFunc("/purge-payments", PurgePaymentsHandler(paymentUsecase))
	mux.HandleFunc("/", HealthCheckHandler)

	server := &http.Server{
		Addr:    ":8000",
		Handler: mux,
	}
	http2.ConfigureServer(server, &http2.Server{})
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

// HANDLERS

func PaymentsSummaryHandler(u *usecase.PaymentUsecase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fromStr := r.URL.Query().Get("from")
		toStr := r.URL.Query().Get("to")

		var fromPtr, toPtr *time.Time
		if fromStr != "" {
			from, err := time.Parse(time.RFC3339, fromStr)
			if err != nil {
				http.Error(w, "Invalid 'from' timestamp", http.StatusBadRequest)
				return
			}
			fromPtr = &from
		}
		if toStr != "" {
			to, err := time.Parse(time.RFC3339, toStr)
			if err != nil {
				http.Error(w, "Invalid 'to' timestamp", http.StatusBadRequest)
				return
			}
			toPtr = &to
		}

		summary, err := u.GetPaymentsSummary(fromPtr, toPtr)
		if err != nil {
			http.Error(w, "Failed to get summary", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(summary)
	}
}

func PurgePaymentsHandler(u *usecase.PaymentUsecase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost && r.Method != http.MethodDelete {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := u.PurgeAll(); err != nil {
			http.Error(w, "Failed to purge data: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}
}

func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
