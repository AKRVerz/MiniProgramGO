package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	httptrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/net/http"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

// chaosConfig holds the current simulation settings, adjustable at runtime
// via the /config endpoint. This lets you demo "normal" vs "degraded" state
// live during the screencast without restarting the service.
type chaosConfig struct {
	mu        sync.RWMutex
	latencyMs int     // extra artificial latency added to every request
	errorRate float64 // probability (0.0 - 1.0) that a request fails with 5xx
}

var chaos = &chaosConfig{}

func (c *chaosConfig) get() (int, float64) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.latencyMs, c.errorRate
}

func (c *chaosConfig) set(latencyMs int, errorRate float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.latencyMs = latencyMs
	c.errorRate = errorRate
}

func main() {
	tracer.Start(
		tracer.WithService("payment-service"),
		tracer.WithEnv("demo"),
	)
	defer tracer.Stop()

	mux := httptrace.NewServeMux() // every route registered here is auto-instrumented

	// /process simulates real payment work: a fake DB call + processing time.
	mux.HandleFunc("/process", handleProcess)

	// /config lets you change latency & error-rate on the fly, e.g.:
	//   curl -X POST "http://localhost:8082/config?latency_ms=800&error_rate=0.3"
	mux.HandleFunc("/config", handleConfig)

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	port := getenv("PORT", "8082")
	log.Printf("payment-service listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func handleProcess(w http.ResponseWriter, r *http.Request) {
	span, ctx := tracer.StartSpanFromContext(r.Context(), "payment.process")
	defer span.Finish()

	latencyMs, errorRate := chaos.get()

	// Child span: pretend DB lookup, so the trace has real depth to inspect.
	dbSpan, _ := tracer.StartSpanFromContext(ctx, "db.query", tracer.ResourceName("SELECT * FROM payments"))
	time.Sleep(time.Duration(10+rand.Intn(20)) * time.Millisecond)
	dbSpan.Finish()

	// Artificial latency to demo the Latency golden signal.
	if latencyMs > 0 {
		time.Sleep(time.Duration(latencyMs) * time.Millisecond)
	}

	// Artificial error injection to demo the Error Rate golden signal.
	if rand.Float64() < errorRate {
		err := fmt.Errorf("simulated downstream payment failure")
		span.SetTag("error", true)
		span.SetTag("error.msg", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "payment processed")
}

func handleConfig(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	latencyMs, _ := strconv.Atoi(q.Get("latency_ms"))
	errorRate, _ := strconv.ParseFloat(q.Get("error_rate"), 64)
	chaos.set(latencyMs, errorRate)
	fmt.Fprintf(w, "config updated: latency_ms=%d error_rate=%.2f\n", latencyMs, errorRate)
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
