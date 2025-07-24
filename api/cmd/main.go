package main

import (
	"api/config"
	"api/usecase"
	"io"
	"log"
	"net/http"

	"golang.org/x/net/http2"
)

func main() {
	config.InitNATS()
	paymentUsecase := usecase.NewPaymentUsecase(config.NatsJS)

	mux := http.NewServeMux()
	mux.HandleFunc("/payments", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		// Leia TODO o body dentro do handler (antes de liberar a resposta)
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		r.Body.Close()
		log.Printf("Payments")

		go paymentUsecase.HandlePaymentRequest(bodyBytes)
		w.WriteHeader(http.StatusAccepted)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		//w.Write([]byte("OK"))
	})

	server := &http.Server{
		Addr:    ":8000",
		Handler: mux,
	}
	http2.ConfigureServer(server, &http2.Server{})
	server.ListenAndServe()
}
