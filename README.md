# Go Highload Microservice

Микросервис на Go 1.22+ с:
- CRUD API для пользователей (`/api/users`)
- конкурентной асинхронной обработкой (audit/notification/error workers)
- rate limiting (`1000 req/s`, `burst 5000`)
- метриками Prometheus (`/metrics`)
- интеграцией с MinIO (S3-совместимое хранилище для audit логов)
- запуском через Docker и docker-compose

## Структура

```text
go-microservice/
├── main.go
├── handlers/
│   ├── user_handler.go
│   └── integration_handler.go
├── services/
│   ├── user_service.go
│   ├── integration_service.go
│   └── audit_service.go
├── models/
│   └── user.go
├── utils/
│   ├── logger.go
│   └── rate_limiter.go
├── metrics/
│   └── prometheus.go
├── go.mod
├── go.sum
├── Dockerfile
├── docker-compose.yml
└── README.md
```

## API

- `GET /api/users` - список пользователей
- `GET /api/users/{id}` - пользователь по ID
- `POST /api/users` - создать пользователя
- `PUT /api/users/{id}` - обновить пользователя
- `DELETE /api/users/{id}` - удалить пользователя
- `GET /api/integration/health` - статус MinIO-интеграции
- `GET /metrics` - метрики Prometheus
- `GET /healthz` - healthcheck

### Пример запроса

```bash
curl -X POST http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{"name":"Ivan","email":"ivan@example.com"}'
```

## Локальный запуск

```bash
go mod tidy
go run ./main.go
```

## Запуск в Docker

```bash
docker-compose up --build
```

## Нагрузочное тестирование (wrk)

```bash
wrk -t12 -c500 -d60s http://localhost:8080/api/users
```

Целевые показатели:
- `RPS > 1000`
- `avg latency < 10ms`
- `errors = 0`

## Метрики Prometheus

```bash
curl http://localhost:8080/metrics
```

Ключевые метрики:
- `http_requests_total`
- `http_request_duration_seconds`
- `http_request_errors_total`
