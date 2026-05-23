# Observability

Backend exposes Prometheus-compatible metrics at:

```text
GET /metrics
```

The request log is emitted as one JSON object per line on stdout. This is suitable for ELK/Filebeat ingestion and includes `request_id`, route, method, status, latency, client IP, user agent, and authenticated user context when available.

Run Prometheus and Grafana with:

```powershell
docker compose -f docker-compose.yml -f docker-compose.observability.yml up
```

Local URLs:

- Prometheus: `http://localhost:9090`
- Grafana: `http://localhost:3000` with `admin` / `admin`
- Backend metrics: `http://localhost:8080/metrics`

Useful Grafana/Prometheus queries:

```promql
sum by (route, status) (rate(digimart_http_requests_total[5m]))
histogram_quantile(0.95, sum by (le, route) (rate(digimart_http_request_duration_seconds_bucket[5m])))
digimart_http_requests_in_flight
digimart_uptime_seconds
```

For ELK, point Filebeat/Logstash at the backend container or process stdout and parse each line as JSON.
