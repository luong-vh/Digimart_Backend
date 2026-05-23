package middleware

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/luong-vh/Digimart_Backend/internal/auth"
)

var (
	metricsStartedAt = time.Now()
	metricsBuckets   = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}
	httpMetrics      = newHTTPMetrics()
)

type routeMetricKey struct {
	Method string
	Route  string
	Status int
}

type durationMetric struct {
	Buckets []uint64
	Count   uint64
	Sum     float64
}

type httpMetricsStore struct {
	mu       sync.Mutex
	Requests map[routeMetricKey]uint64
	Duration map[routeMetricKey]*durationMetric
	InFlight uint64
}

func newHTTPMetrics() *httpMetricsStore {
	return &httpMetricsStore{
		Requests: make(map[routeMetricKey]uint64),
		Duration: make(map[routeMetricKey]*durationMetric),
	}
}

func (m *httpMetricsStore) startRequest() {
	m.mu.Lock()
	m.InFlight++
	m.mu.Unlock()
}

func (m *httpMetricsStore) finishRequest(key routeMetricKey, seconds float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.InFlight > 0 {
		m.InFlight--
	}
	m.Requests[key]++

	duration, ok := m.Duration[key]
	if !ok {
		duration = &durationMetric{Buckets: make([]uint64, len(metricsBuckets))}
		m.Duration[key] = duration
	}
	duration.Count++
	duration.Sum += seconds
	for i, bucket := range metricsBuckets {
		if seconds <= bucket {
			duration.Buckets[i]++
		}
	}
}

// RequestID attaches a stable request id to each request for log correlation.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.NewString()
		}

		c.Set("requestID", requestID)
		c.Writer.Header().Set("X-Request-ID", requestID)
		c.Next()
	}
}

// RequestLogger writes structured JSON access logs suitable for ELK ingestion.
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		if c.FullPath() == "/metrics" {
			return
		}

		status := c.Writer.Status()
		route := normalizedRoute(c)
		latency := time.Since(start)
		entry := map[string]interface{}{
			"timestamp":  time.Now().UTC().Format(time.RFC3339Nano),
			"level":      logLevel(status),
			"message":    "http_request",
			"request_id": requestID(c),
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"route":      route,
			"status":     status,
			"latency_ms": float64(latency.Microseconds()) / 1000,
			"client_ip":  c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
		}

		if rawQuery := c.Request.URL.RawQuery; rawQuery != "" {
			entry["query"] = rawQuery
		}
		if len(c.Errors) > 0 {
			entry["error"] = c.Errors.String()
		}
		if user, ok := c.Get("authUser"); ok {
			if authUser, ok := user.(auth.AuthUser); ok {
				entry["user_id"] = authUser.ID
				entry["user_role"] = authUser.Role
			}
		}

		payload, err := json.Marshal(entry)
		if err != nil {
			log.Printf(`{"level":"error","message":"request_log_marshal_failed","error":%q}`, err.Error())
			return
		}
		log.Println(string(payload))
	}
}

// Metrics records HTTP request counters and latency histograms.
func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.FullPath() == "/metrics" || c.Request.URL.Path == "/metrics" {
			c.Next()
			return
		}

		start := time.Now()
		httpMetrics.startRequest()
		c.Next()

		httpMetrics.finishRequest(routeMetricKey{
			Method: c.Request.Method,
			Route:  normalizedRoute(c),
			Status: c.Writer.Status(),
		}, time.Since(start).Seconds())
	}
}

// MetricsHandler exposes Prometheus-compatible metrics without external dependencies.
func MetricsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		c.String(http.StatusOK, renderPrometheusMetrics())
	}
}

func renderPrometheusMetrics() string {
	httpMetrics.mu.Lock()
	defer httpMetrics.mu.Unlock()

	var builder strings.Builder
	builder.WriteString("# HELP digimart_uptime_seconds Application uptime in seconds.\n")
	builder.WriteString("# TYPE digimart_uptime_seconds gauge\n")
	builder.WriteString(fmt.Sprintf("digimart_uptime_seconds %.0f\n", time.Since(metricsStartedAt).Seconds()))
	builder.WriteString("# HELP digimart_http_requests_in_flight Current in-flight HTTP requests.\n")
	builder.WriteString("# TYPE digimart_http_requests_in_flight gauge\n")
	builder.WriteString(fmt.Sprintf("digimart_http_requests_in_flight %d\n", httpMetrics.InFlight))
	builder.WriteString("# HELP digimart_http_requests_total Total HTTP requests by method, route, and status.\n")
	builder.WriteString("# TYPE digimart_http_requests_total counter\n")

	keys := sortedMetricKeys(httpMetrics.Requests)
	for _, key := range keys {
		builder.WriteString(fmt.Sprintf(
			"digimart_http_requests_total{method=%q,route=%q,status=%q} %d\n",
			escapeLabel(key.Method),
			escapeLabel(key.Route),
			strconv.Itoa(key.Status),
			httpMetrics.Requests[key],
		))
	}

	builder.WriteString("# HELP digimart_http_request_duration_seconds HTTP request latency histogram.\n")
	builder.WriteString("# TYPE digimart_http_request_duration_seconds histogram\n")
	for _, key := range sortedDurationKeys(httpMetrics.Duration) {
		duration := httpMetrics.Duration[key]
		for i, bucket := range metricsBuckets {
			builder.WriteString(fmt.Sprintf(
				"digimart_http_request_duration_seconds_bucket{method=%q,route=%q,status=%q,le=%q} %d\n",
				escapeLabel(key.Method),
				escapeLabel(key.Route),
				strconv.Itoa(key.Status),
				strconv.FormatFloat(bucket, 'f', -1, 64),
				duration.Buckets[i],
			))
		}
		builder.WriteString(fmt.Sprintf(
			"digimart_http_request_duration_seconds_bucket{method=%q,route=%q,status=%q,le=%q} %d\n",
			escapeLabel(key.Method),
			escapeLabel(key.Route),
			strconv.Itoa(key.Status),
			"+Inf",
			duration.Count,
		))
		builder.WriteString(fmt.Sprintf(
			"digimart_http_request_duration_seconds_sum{method=%q,route=%q,status=%q} %s\n",
			escapeLabel(key.Method),
			escapeLabel(key.Route),
			strconv.Itoa(key.Status),
			strconv.FormatFloat(duration.Sum, 'f', -1, 64),
		))
		builder.WriteString(fmt.Sprintf(
			"digimart_http_request_duration_seconds_count{method=%q,route=%q,status=%q} %d\n",
			escapeLabel(key.Method),
			escapeLabel(key.Route),
			strconv.Itoa(key.Status),
			duration.Count,
		))
	}

	return builder.String()
}

func normalizedRoute(c *gin.Context) string {
	if route := c.FullPath(); route != "" {
		return route
	}
	return "unmatched"
}

func requestID(c *gin.Context) string {
	if value, ok := c.Get("requestID"); ok {
		if requestID, ok := value.(string); ok {
			return requestID
		}
	}
	return ""
}

func logLevel(status int) string {
	switch {
	case status >= http.StatusInternalServerError:
		return "error"
	case status >= http.StatusBadRequest:
		return "warn"
	default:
		return "info"
	}
}

func escapeLabel(value string) string {
	return strings.NewReplacer("\\", "\\\\", "\n", "\\n", "\"", "\\\"").Replace(value)
}

func sortedMetricKeys(values map[routeMetricKey]uint64) []routeMetricKey {
	keys := make([]routeMetricKey, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sortMetricKeys(keys)
	return keys
}

func sortedDurationKeys(values map[routeMetricKey]*durationMetric) []routeMetricKey {
	keys := make([]routeMetricKey, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sortMetricKeys(keys)
	return keys
}

func sortMetricKeys(keys []routeMetricKey) {
	sort.Slice(keys, func(i, j int) bool {
		left := keys[i]
		right := keys[j]
		if left.Route != right.Route {
			return left.Route < right.Route
		}
		if left.Method != right.Method {
			return left.Method < right.Method
		}
		return left.Status < right.Status
	})
}
