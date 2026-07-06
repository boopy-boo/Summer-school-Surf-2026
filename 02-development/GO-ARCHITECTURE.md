# Архитектура и стек Go-приложения: Гончарная мастерская «Глина»
> Рекомендации по построению клиентского API на Go

---

## 1. Роль Go-приложения в системе

```
┌─────────────┐     ┌─────────────────────┐     ┌─────────────────┐
│  Мобильное  │◄───►│  Go API (BFF)       │◄───►│  Существующий   │
│  приложение │     │  • адаптер          │     │  бэкенд         │
│  (Flutter/  │     │  • агрегация        │     │  (black-box)    │
│   React    │     │  • кэширование      │     │                 │
│   Native)  │     │  • валидация        │     │                 │
└─────────────┘     └─────────────────────┘     └─────────────────┘
                            │
                            ▼
                    ┌───────────────┐
                    │  Redis (OTP,  │
                    │  сессии, кэш) │
                    └───────────────┘
```

Go-сервис выступает **BFF (Backend for Frontend)** — адаптер между мобильным приложением и существующей инфраструктурой. Не дублирует бизнес-логику бэкенда, но берёт на себя:
- Агрегацию данных для мобильных экранов
- Кэширование справочников (мастера, программы)
- Управление сессиями и OTP
- Валидацию входящих запросов
- Форматирование ответов под мобильное приложение

---

## 2. Стек технологий

### Ядро

| Категория | Рекомендация | Альтернатива | Почему |
|-----------|--------------|--------------|--------|
| **HTTP Router** | `go-chi/chi` v5 | `echo`, `gin`, `stdlib` | Стандартные `http.Handler`, лёгкий, middleware-цепочки, production-ready |
| **Валидация** | `go-playground/validator` v10 | ручная | Struct tags, кастомные правила, перевод ошибок |
| **JSON** | `stdlib encoding/json` | `jsoniter`, `sonic` | Достаточно для MVP; заменить если узкое место |
| **Конфигурация** | `ilyakaznacheev/cleanenv` | `spf13/viper` | Struct-based, env-файлы, простой и предсказуемый |
| **Логирование** | `uber-go/zap` | `rs/zerolog`, `slog` | Производительность, structured logging, стандарт индустрии |
| **Dependency Injection** | Ручной (функции-конструкторы) | `uber-go/fx` | Для MVP проще; `fx` при >10 сервисов |

### Инфраструктура

| Категория | Рекомендация | Альтернатива | Почему |
|-----------|--------------|--------------|--------|
| **Кэш / OTP Store** | `redis/go-redis` v9 | PostgreSQL, in-memory | TTL для OTP, rate limiting, кэш справочников |
| **БД (если нужна)** | `jackc/pgx` v5 + `sqlc` | `gorm`, `bun` | pgx — быстрый драйвер; sqlc — типобезопасный SQL |
| **Миграции** | `golang-migrate/migrate` | `pressly/goose` | CLI + Go API, PostgreSQL/MySQL, проверен временем |
| **HTTP Client** | `go-resty/resty` v2 | `stdlib net/http` | Retry, timeout, middleware, удобная работа с JSON |
| **Graceful Shutdown** | `stdlib` (`signal.NotifyContext`) | — | Нативный Go, никаких зависимостей |

### DevEx / CI

| Категория | Рекомендация | Альтернатива |
|-----------|--------------|--------------|
| **Тесты** | `stretchr/testify` + `httptest` | — |
| **Моки** | `vektra/mockery` + `mockery` | ручные интерфейсы |
| **Линтер** | `golangci-lint` | — |
| **API Docs** | `swaggo/swag` | ручной OpenAPI |
| **Контейнеризация** | Docker + Docker Compose | — |

---

## 3. Архитектура: Clean Architecture (Hexagonal)

### Принципы
1. **Зависимости внутрь** — внутренние слои не знают о внешних
2. **Бизнес-логика изолирована** — не зависит от фреймворков и БД
3. **Порты и адаптеры** — интерфейсы определяют контракты, реализации подменяются
4. **Тестируемость** — моки через интерфейсы

### Слои (сверху вниз)

| Слой | Ответственность | Примеры |
|------|-----------------|---------|
| **Handler / Transport** | HTTP-вход, JSON сериализация, валидация входа | `chi.Router`, request/response DTO, middleware |
| **Service / Usecase** | Бизнес-логика, оркестрация, политики | расчёт max_seats, проверка политик отмены |
| **Repository / Port** | Абстракция хранилища | интерфейс `SlotRepo`, `BookingRepo` |
| **Adapter / Infrastructure** | Реализация репозиториев, HTTP-клиенты к бэкенду | `BackendSlotClient`, `RedisOtpStore` |

### Правило зависимостей
```
Handler ──► Service ──► Repository (интерфейс)
  ▲                       ▲
  │                       │
  └── Adapter ◄───────────┘
```

---

## 4. Структура проекта

```
pottery-api/
├── cmd/
│   └── api/
│       └── main.go                 # Точка входа, wiring DI
├── internal/
│   ├── config/
│   │   └── config.go               # Cleanenv struct, валидация конфига
│   ├── domain/                     # Чистый домен (zero dependencies)
│   │   ├── slot.go                 # Slot, Program, Master entities
│   │   ├── booking.go              # Booking entity, статусы, инварианты
│   │   ├── client.go               # Client entity
│   │   └── errors.go               # Доменные ошибки (ErrNotFound, ErrConflict)
│   ├── service/                    # Бизнес-логика (usecase layer)
│   │   ├── booking_service.go      # BookingUsecase, политики отмены
│   │   ├── slot_service.go         # SlotUsecase, агрегация данных
│   │   └── auth_service.go         # OTP, JWT, сессии
│   ├── repository/                 # Интерфейсы (ports)
│   │   ├── slot_repo.go            # interface SlotReader
│   │   ├── booking_repo.go         # interface BookingStore
│   │   ├── otp_store.go            # interface OTPStore
│   │   └── backend_client.go       # interface BackendClient
│   ├── adapter/                    # Реализации (adapters)
│   │   ├── http/
│   │   │   └── backend_client.go   # REST-клиент к существующему бэкенду
│   │   ├── redis/
│   │   │   └── otp_store.go        # Хранение OTP в Redis с TTL
│   │   └── memory/
│   │       └── slot_cache.go       # In-memory кэш справочников
│   ├── handler/                    # HTTP handlers (transport)
│   │   ├── slot_handler.go
│   │   ├── booking_handler.go
│   │   ├── auth_handler.go
│   │   └── middleware/
│   │       ├── auth.go             # JWT middleware
│   │       ├── request_id.go
│   │       └── logger.go
│   └── http/                       # App server setup
│       └── server.go               # chi.Router, graceful shutdown
├── pkg/
│   └── validator/                  # Обёртка над go-playground/validator
├── api/                            # OpenAPI / proto (автогенерация)
│   └── openapi.yaml
├── migrations/
│   └── 001_init.up.sql             # Если нужна локальная БД для сессий
├── docker-compose.yml
├── Dockerfile
├── Makefile
├── .env.example
└── go.mod
```

### Ключевые решения по структуре

| Решение | Обоснование |
|---------|-------------|
| `internal/` | Запрет импорта извне, чёткая граница приложения |
| `domain/` | Zero dependencies — можно выделить в отдельный модуль |
| `repository/` — интерфейсы | Инверсия зависимостей, моки для тестов |
| `adapter/` — реализации | Подмена Redis на in-memory для интеграционных тестов |
| `service/` | Оркестрация, политики (граница ранней/поздней отмены и т.д.) |

---

## 5. Примеры кода

### 5.1 Доменная сущность

```go
// internal/domain/booking.go
package domain

import (
    "errors"
    "time"
)

var (
    ErrBookingNotFound     = errors.New("booking not found")
    ErrSlotFullyBooked     = errors.New("no seats available")
    ErrLateCancel          = errors.New("late cancellation not allowed")
    ErrSlotCancelled       = errors.New("slot cancelled by workshop")
)

type BookingStatus string

const (
    BookingActive           BookingStatus = "active"
    BookingCancelled        BookingStatus = "cancelled"
    BookingLateCancel       BookingStatus = "late_cancel"
    BookingWorkshopCancelled BookingStatus = "workshop_cancelled"
)

type Booking struct {
    ID                string
    SlotID            string
    ClientID          string
    SeatsCount        int
    RentalCount       int
    PriceTotal        float64
    Status            BookingStatus
    CancellationReason string
    CreatedAt         time.Time
    CancelledAt       *time.Time
}

// MaxSeats — лимит мест на одну бронь
func MaxSeats(freeSeats, capacityCap int) int {
    const bookingLimit = 3
    max := freeSeats
    if capacityCap < max {
        max = capacityCap
    }
    if bookingLimit < max {
        max = bookingLimit
    }
    return max
}

// CanCancel — проверяет, можно ли отменить бронь без последствий
func (b Booking) CanCancel(now, slotStart time.Time) bool {
    if b.Status != BookingActive {
        return false
    }
    return now.Before(slotStart.Add(-2 * time.Hour)) // ≥2 часов
}
```

### 5.2 Репозиторий (интерфейс / port)

```go
// internal/repository/slot_repo.go
package repository

import (
    "context"
    "time"

    "pottery-api/internal/domain"
)

type SlotReader interface {
    List(ctx context.Context, filter SlotFilter) ([]domain.Slot, int, error)
    GetByID(ctx context.Context, id string) (*domain.Slot, error)
}

type SlotFilter struct {
    DateFrom      *time.Time
    DateTo        *time.Time
    ProgramTypes  []string
    MasterIDs     []string
    OnlyAvailable bool
    Limit         int
    Offset        int
}
```

### 5.3 Адаптер (реализация)

```go
// internal/adapter/http/backend_client.go
package httpadapter

import (
    "context"
    "fmt"
    "net/http"
    "time"

    "github.com/go-resty/resty/v2"
    "pottery-api/internal/domain"
    "pottery-api/internal/repository"
)

type BackendClient struct {
    client *resty.Client
}

func NewBackendClient(baseURL string, timeout time.Duration) *BackendClient {
    return &BackendClient{
        client: resty.New().
            SetBaseURL(baseURL).
            SetTimeout(timeout).
            SetHeader("Content-Type", "application/json"),
    }
}

func (c *BackendClient) ListSlots(ctx context.Context, filter repository.SlotFilter) ([]domain.Slot, int, error) {
    req := c.client.R().SetContext(ctx)
    
    if filter.DateFrom != nil {
        req.SetQueryParam("date_from", filter.DateFrom.Format(time.RFC3339))
    }
    if filter.DateTo != nil {
        req.SetQueryParam("date_to", filter.DateTo.Format(time.RFC3339))
    }
    // ... остальные фильтры

    var resp SlotListResponse
    r, err := req.SetResult(&resp).Get("/slots")
    if err != nil {
        return nil, 0, fmt.Errorf("backend request failed: %w", err)
    }
    if r.StatusCode() != http.StatusOK {
        return nil, 0, fmt.Errorf("backend error: %s", r.Status())
    }

    // map response to domain
    slots := make([]domain.Slot, len(resp.Items))
    for i, item := range resp.Items {
        slots[i] = mapSlotItem(item)
    }
    return slots, resp.Total, nil
}
```

### 5.4 Сервис (бизнес-логика)

```go
// internal/service/booking_service.go
package service

import (
    "context"
    "fmt"
    "time"

    "pottery-api/internal/domain"
    "pottery-api/internal/repository"
)

type BookingUsecase struct {
    bookingRepo repository.BookingStore
    slotReader  repository.SlotReader
    otpStore    repository.OTPStore
    clock       func() time.Time
}

func NewBookingUsecase(
    br repository.BookingStore,
    sr repository.SlotReader,
    os repository.OTPStore,
) *BookingUsecase {
    return &BookingUsecase{
        bookingRepo: br,
        slotReader:  sr,
        otpStore:    os,
        clock:       time.Now,
    }
}

func (u *BookingUsecase) Create(ctx context.Context, clientID string, req BookingRequest) (*domain.Booking, error) {
    slot, err := u.slotReader.GetByID(ctx, req.SlotID)
    if err != nil {
        return nil, fmt.Errorf("get slot: %w", err)
    }

    // Доменная валидация
    maxSeats := domain.MaxSeats(slot.FreeSeats, slot.Program.CapacityCap)
    if len(req.Seats) > maxSeats {
        return nil, domain.ErrSlotFullyBooked
    }

    rentalCount := countRental(req.Seats)
    if rentalCount > slot.FreeRentalKits {
        return nil, domain.ErrSlotFullyBooked
    }

    if slot.Status == domain.SlotCancelled {
        return nil, domain.ErrSlotCancelled
    }

    // Создание — делегируем бэкенду ( black-box источник истины )
    booking, err := u.bookingRepo.Create(ctx, clientID, req)
    if err != nil {
        return nil, fmt.Errorf("create booking: %w", err)
    }

    return booking, nil
}

func (u *BookingUsecase) Cancel(ctx context.Context, clientID, bookingID string) error {
    booking, err := u.bookingRepo.GetByID(ctx, clientID, bookingID)
    if err != nil {
        return fmt.Errorf("get booking: %w", err)
    }

    slot, err := u.slotReader.GetByID(ctx, booking.SlotID)
    if err != nil {
        return fmt.Errorf("get slot: %w", err)
    }

    now := u.clock()
    if !booking.CanCancel(now, slot.StartAt) {
        // Поздняя отмена — сервер решает, мы только передаём
        return u.bookingRepo.CancelLate(ctx, clientID, bookingID)
    }

    return u.bookingRepo.Cancel(ctx, clientID, bookingID)
}
```

### 5.5 Handler

```go
// internal/handler/booking_handler.go
package handler

import (
    "errors"
    "net/http"

    "github.com/go-chi/chi/v5"
    "pottery-api/internal/domain"
    "pottery-api/internal/service"
)

type BookingHandler struct {
    usecase *service.BookingUsecase
}

func NewBookingHandler(uc *service.BookingUsecase) *BookingHandler {
    return &BookingHandler{usecase: uc}
}

func (h *BookingHandler) RegisterRoutes(r chi.Router) {
    r.Get("/bookings", h.List)
    r.Post("/bookings", h.Create)
    r.Get("/bookings/{bookingId}", h.Get)
    r.Delete("/bookings/{bookingId}", h.Cancel)
}

func (h *BookingHandler) Create(w http.ResponseWriter, r *http.Request) {
    var req service.BookingRequest
    if err := decodeJSON(r, &req); err != nil {
        respondError(w, http.StatusBadRequest, "invalid JSON")
        return
    }

    clientID := r.Context().Value("client_id").(string)
    booking, err := h.usecase.Create(r.Context(), clientID, req)
    if err != nil {
        switch {
        case errors.Is(err, domain.ErrSlotFullyBooked):
            respondError(w, http.StatusConflict, "no seats available")
        case errors.Is(err, domain.ErrSlotCancelled):
            respondError(w, http.StatusGone, "slot cancelled by workshop")
        default:
            respondError(w, http.StatusInternalServerError, "internal error")
        }
        return
    }

    respondJSON(w, http.StatusCreated, booking)
}
```

### 5.6 Wiring (main.go)

```go
// cmd/api/main.go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"

    "pottery-api/internal/adapter/http"
    "pottery-api/internal/adapter/redis"
    "pottery-api/internal/config"
    "pottery-api/internal/handler"
    "pottery-api/internal/httpserver"
    "pottery-api/internal/service"
)

func main() {
    cfg, err := config.Load()
    if err != nil {
        log.Fatalf("load config: %v", err)
    }

    // Adapters
    backend := httpadapter.NewBackendClient(cfg.BackendURL, cfg.BackendTimeout)
    redisClient := redis.NewOTPStore(cfg.RedisAddr, cfg.RedisPass)

    // Services
    slotSvc := service.NewSlotService(backend)
    bookingSvc := service.NewBookingUsecase(backend, backend, redisClient)

    // Handlers
    slotH := handler.NewSlotHandler(slotSvc)
    bookingH := handler.NewBookingHandler(bookingSvc)
    authH := handler.NewAuthHandler(redisClient, cfg.JWTSecret)

    // Server
    srv := httpserver.New(
        httpserver.WithHandler(slotH),
        httpserver.WithHandler(bookingH),
        httpserver.WithHandler(authH),
        httpserver.WithAddr(cfg.HTTPAddr),
    )

    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    log.Printf("starting server on %s", cfg.HTTPAddr)
    if err := srv.Run(ctx); err != nil {
        log.Fatalf("server error: %v", err)
    }
}
```

---

## 6. Конфигурация (cleanenv)

```go
// internal/config/config.go
package config

import (
    "time"

    "github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
    HTTPAddr       string        `env:"HTTP_ADDR" env-default:":8080"`
    BackendURL     string        `env:"BACKEND_URL" env-required:"true"`
    BackendTimeout time.Duration `env:"BACKEND_TIMEOUT" env-default:"5s"`
    RedisAddr      string        `env:"REDIS_ADDR" env-default:"localhost:6379"`
    RedisPass      string        `env:"REDIS_PASS"`
    JWTSecret      string        `env:"JWT_SECRET" env-required:"true"`
    OTPTimeout     time.Duration `env:"OTP_TIMEOUT" env-default:"5m"`
}

func Load() (*Config, error) {
    var cfg Config
    if err := cleanenv.ReadEnv(&cfg); err != nil {
        return nil, err
    }
    return &cfg, nil
}
```

---

## 7. Сравнение альтернатив

### Router: chi vs echo vs gin

| Критерий | chi | echo | gin |
|----------|-----|------|-----|
| `http.Handler` совместимость | ✅ Нативная | ⚠️ Свой контекст | ⚠️ Свой контекст |
| Middleware-цепочки | ✅ | ✅ | ✅ |
| Производительность | Быстрый | Быстрый | Очень быстрый |
| Размер | ~800 KB | ~2 MB | ~2 MB |
| **Рекомендация** | **chi** — стандартный интерфейс, легко тестировать | — | — |

### ORM/Database: sqlc vs gorm

| Критерий | sqlc + pgx | GORM |
|----------|-----------|------|
| Типобезопасность | ✅ SQL → Go код | ⚠️ Интерфейс `interface{}` |
| Производительность | ✅ Быстрее | ⚠️ Рефлексия |
| Сложные запросы | ✅ Чистый SQL | ✅ API |
| Миграции | Отдельно (golang-migrate) | Встроенные |
| **Рекомендация** | **sqlc + pgx** — для нашего случая (генерация read моделей) | — |

### Логи: zap vs zerolog vs slog

| Критерий | zap | zerolog | slog |
|----------|-----|---------|------|
| Производительность | Очень высокая | Очень высокая | Хорошая |
| Обогащение контекста | Ext.Y development | ⚠️ Меньше примеров | ✅ Стандарт |
| **Рекомендация** | **zap** — зрелый, много адаптеров | — | **slog** для новых проектов (Go 1.21+) |

### DI: ручной vs uber-go/fx

| Критерий | Ручной | uber-go/fx |
|----------|--------|------------|
| Явность | ✅ Всё видно в main | ⚠️ Магия reflection |
| Startup time | Мгновенный | Небольшой overhead |
| **Рекомендация** | **Ручной** для MVP (5-10 сервисов) | fx при >15 компонентов |

---

## 8. План внедрения

### Этап 0: Скелет (1-2 дня)
1. `go mod init`, структура папок
2. chi router + graceful shutdown
3. cleanenv конфигурация
4. zap логирование + middleware (request_id, логирование запросов)

### Этап 1: Auth (2-3 дня)
1. Redis-адаптер для OTP (TTL 5 мин, лимиты)
2. POST /auth/otp/send, POST /auth/otp/verify
3. JWT middleware (access 24ч, refresh 30дней)
4. POST /auth/refresh

### Этап 2: Slots read-only (2-3 дня)
1. Backend HTTP-client (resty)
2. GET /slots (фильтры, пагинация)
3. GET /slots/{id} (полная карточка)
4. GET /masters (кэш в Redis на 1 час)
5. Валидация входящих параметров

### Этап 3: Bookings (3-4 дня)
1. POST /bookings (доменная валидация + проксирование на бэкенд)
2. GET /bookings, GET /bookings/{id}
3. DELETE /bookings/{id} (логика ранней/поздней отмены)
4. Обработка ошибок бэкенда (409, 410)

### Этап 4: Profile + Quality (2 дня)
1. GET /profile, PATCH /profile, DELETE /profile
2. Тесты: unit ( testify + моки ), integration (httptest + Redis container)
3. Swagger документация (swaggo)
4. Docker + docker-compose (Redis + app)

---

## 9. Docker Compose для разработки

```yaml
# docker-compose.yml
version: "3.8"
services:
  app:
    build: .
    ports:
      - "8080:8080"
    environment:
      - HTTP_ADDR=:8080
      - BACKEND_URL=http://existing-backend:8080
      - REDIS_ADDR=redis:6379
      - JWT_SECRET=dev-secret-change-me
    depends_on:
      - redis

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

  # Для интеграционных тестов
  backend-mock:
    image: wiremock/wiremock:3
    ports:
      - "8081:8080"
    volumes:
      - ./test/wiremock:/home/wiremock
```

---

## 10. Makefile

```makefile
.PHONY: run build test lint migrate swagger

run:
	go run ./cmd/api

build:
	go build -o bin/api ./cmd/api

test:
	go test -race -cover ./...

lint:
	golangci-lint run

swagger:
	swag init -g cmd/api/main.go

migrate-up:
	migrate -path migrations -database "postgres://...?sslmode=disable" up
```

---

## 11. Итоговый выбор стека

```go
// go.mod (основные зависимости)

require (
    github.com/go-chi/chi/v5                    // HTTP router
    github.com/go-chi/chi/v5/middleware
    github.com/go-playground/validator/v10      // Валидация
    github.com/go-resty/resty/v2                // HTTP-клиент
    github.com/redis/go-redis/v9                // Redis
    github.com/golang-jwt/jwt/v5                // JWT
    go.uber.org/zap                             // Логирование
    github.com/ilyakaznacheev/cleanenv          // Конфигурация
)
```

| Компонент | Выбор |
|-----------|-------|
| **Router** | `chi/v5` |
| **Валидация** | `validator/v10` |
| **HTTP-клиент** | `resty/v2` |
| **Кэш/OTP** | `redis/go-redis/v9` |
| **Auth** | `jwt/v5` + ручной OTP |
| **Логи** | `zap` |
| **Config** | `cleanenv` |
| **Архитектура** | **Clean Architecture (Hexagonal)** |
| **DI** | Ручной (функции-конструкторы) |
| **Тесты** | `testify` + `httptest` |
| **Docs** | `swaggo/swag` |
| **Контейнеры** | Docker + Docker Compose |

---

> **Следующий шаг:** Создать скелет проекта (`cmd/api/main.go` + структура папок + `go.mod`) по этой архитектуре?