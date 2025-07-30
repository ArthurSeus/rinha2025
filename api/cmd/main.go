package main

import (
	"api/config"
	"api/usecase"
	"io"
	"net/http"

	"golang.org/x/net/http2"
)

func main() {
	nats := config.JetStreamInit()

	paymentUsecase := usecase.NewPaymentUsecase(nats)

	mux := http.NewServeMux()
	mux.HandleFunc("/payments", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		r.Body.Close()

		go paymentUsecase.HandlePaymentRequest(bodyBytes)
		w.WriteHeader(http.StatusAccepted)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	mux.HandleFunc("/payments-summary", func(w http.ResponseWriter, r *http.Request) {
		targetURL := "http://payment-worker:8000/payments-summary" + "?" + r.URL.RawQuery
		resp, err := http.Get(targetURL)
		if err != nil {
			http.Error(w, "Failed to get summary: "+err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	})

	mux.HandleFunc("/purge-payments", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost && r.Method != http.MethodDelete {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		client := &http.Client{}
		req, err := http.NewRequest(r.Method, "http://payment-worker:8000/purge-payments", nil)
		if err != nil {
			http.Error(w, "Failed to build request: "+err.Error(), http.StatusInternalServerError)
			return
		}

		resp, err := client.Do(req)
		if err != nil {
			http.Error(w, "Failed to purge payments: "+err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	})

	server := &http.Server{
		Addr:    ":8000",
		Handler: mux,
	}
	http2.ConfigureServer(server, &http2.Server{})
	server.ListenAndServe()
}
