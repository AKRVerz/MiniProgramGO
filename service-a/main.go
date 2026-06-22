package main

import (
	"io"
	"log"
	"net/http"
	"os"

	httptrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/net/http"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

// client is wrapped so that every outgoing call automatically injects the
// trace headers payment-service needs to join the same trace
// (this is what makes the trace "distributed" across two processes).
var client = httptrace.WrapClient(&http.Client{})

func main() {
	tracer.Start(
		tracer.WithService("checkout-service"),
		tracer.WithEnv("demo"),
	)
	defer tracer.Stop()

	mux := httptrace.NewServeMux()
	mux.HandleFunc("/checkout", handleCheckout)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	port := getenv("PORT", "8081")
	log.Printf("checkout-service listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func handleCheckout(w http.ResponseWriter, r *http.Request) {
	paymentURL := getenv("PAYMENT_SERVICE_URL", "http://localhost:8082") + "/process"

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, paymentURL, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
