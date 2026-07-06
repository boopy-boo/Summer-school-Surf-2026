# План реализации Go Back-end (BFF)
> Итеративный чеклист разработки клиентского API для «Гончарной мастерской»

---

## Как пользоваться планом

- Каждая **итерация** — законченный инкремент, который можно собрать и протестировать
- Внутри итерации задачи идут в порядке выполнения
- Отмечайте `[x]` по мере выполнения; при блокере — оставляйте комментарий
- Оценки в днях указаны для 1 разработчика среднего уровня (Go, знаком с clean architecture)

---

## Итерация 0: Инфраструктура и скелет проекта
> Цель: `go run ./cmd/api` поднимает HTTP-сервер на порту 8080 с health-check, graceful shutdown и структурированным логированием.  
> Длительность: **1–2 дня**

### 0.1 Инициализация репозитория
- [ ] `go mod init github.com/pottery-api`
- [ ] Настроить `.gitignore` (bin/, .env, vendor/, coverage.out)
- [ ] Настроить `Makefile` с целями: run, build, test, lint, tidy
- [ ] Установить и настроить `golangci-lint` (`.golangci.yml`)

### 0.2 Структура папок
- [ ] Создать `cmd/api/main.go` (точка входа)
- [ ] Создать `internal/config/config.go` (cleanenv struct)
- [ ] Создать `internal/domain/` — пустые файлы `errors.go`, `slot.go`, `booking.go`, `client.go`
- [ ] Создать `internal/repository/` — пустые интерфейсы
- [ ] Создать `internal/service/` — пустые конструкторы
- [ ] Создать `internal/handler/` — пустые handler-ы
- [ ] Создать `internal/adapter/http/` — заглушка backend client
- [ ] Создать `internal/adapter/redis/` — заглушка redis client
- [ ] Создать `pkg/validator/validator.go` — обёртка над `go-playground/validator`

### 0.3 Зависимости
- [ ] `go get github.com/go-chi/chi/v5`
- [ ] `go get github.com/go-playground/validator/v10`
- [ ] `go get github.com/go-resty/resty/v2`
- [ ] `go get github.com/redis/go-redis/v9`
- [ ] `go get github.com/golang-jwt/jwt/v5`
- [ ] `go get go.uber.org/zap`
- [ ] `go get github.com/ilyakaznacheev/cleanenv`
- [ ] `go get github.com/stretchr/testify`

### 0.4 Конфигурация
- [ ] Определить `Config` struct (HTTPAddr, BackendURL, BackendTimeout, RedisAddr, RedisPass, JWTSecret, OTPTimeout, Environment)
- [ ] Создать `.env.example`
- [ ] Валидация: `BackendURL` и `JWTSecret` — required; остальные — default

### 0.5 HTTP-сервер (скелет)
- [ ] `internal/http/server.go` — `Server` struct, `ListenAndServe`, graceful shutdown via `signal.NotifyContext`
- [ ] chi router с `/health` → `{"status":"ok"}`
- [ ] Middleware: `Recoverer`, `RequestID`, `Logger` (zap)
- [ ] Handler интерфейс `RouteRegistrar` (`RegisterRoutes(chi.Router)`)
- [ ] Wiring в `main.go`: config → router → server → run

### 0.6 Docker
- [ ] `Dockerfile` (multi-stage: builder + alpine)
- [ ] `docker-compose.yml`: app + redis services
- [ ] Проверка: `docker compose up --build` → ответ на `localhost:8080/health`

### 0.7 Логирование
- [ ] Инициализация zap (production/development в зависимости от env)
- [ ] Middleware логирования запросов: method, path, status, duration, request_id
- [ ] Контекстный логгер: `logger.FromContext(ctx)`

### 0.8 CI/CD (опционально, но желательно)
- [ ] GitHub Actions / GitLab CI: lint, test, build

**Критерий приёмки Итерации 0**
```bash
curl http://localhost:8080/health
# {"status":"ok"}
docker compose logs app | grep "starting server"
golangci-lint run # 0 ошибок
```

---

## Итерация 1: Auth / OTP / JWT
> Цель: Рабочий flow «телефон → SMS код → JWT → refresh». Ограничения по rate limiting и TTL.  
> Длительность: **3–4 дня**

### 1.1 Домен: клиент и авторизация
- [ ] `internal/domain/client.go` — `Client` struct (`ID`, `Name`, `Phone`, `CreatedAt`)
- [ ] `internal/domain/errors.go` — `ErrNotFound`, `ErrUnauthorized`, `ErrTooManyRequests`, `ErrInvalidOTP`
- [ ] `internal/domain/otp.go` — `OTP` struct, `OTPStore` interface (generate, verify, attempts, ttl)

### 1.2 Redis адаптер для OTP
- [ ] `internal/adapter/redis/otp_store.go` — реализация `OTPStore`
- [ ] `SETEX` для хранения кода (TTL 5 мин)
- [ ] `INCR` / `EXPIRE` для счётчика попыток (max 3)
- [ ] `INCR` / `EXPIRE` для rate limit отправки (1 раз в 60 сек)
- [ ] `INCR` / `EXPIRE` для rate limit brute-force (5 неудач в час → блок 30 мин) (NFR-11)
- [ ] Unit-тесты с `miniredis`

### 1.3 JWT механизм
- [ ] `internal/service/auth/jwt.go` — генерация access (24ч) и refresh (30 дней)
- [ ] Валидация access token (middleware)
- [ ] `Refresh` endpoint: exchange refresh → new pair
- [ ] Хранение refresh token в Redis (whitelist, возможность инвалидации при logout)

### 1.4 Auth service
- [ ] `internal/service/auth_service.go` — `AuthService`
- [ ] `SendOTP(phone)` — генерация 6-значного кода, rate limiting, вызов SMS-шлюза (заглушка)
- [ ] `VerifyOTP(phone, code, name)` — проверка, создание/получение клиента, выдача JWT
- [ ] `Logout(userID)` — инвалидация refresh token в Redis
- [ ] Заглушка SMS-шлюза: `internal/adapter/sms/sms_gateway.go` (interface + mock)

### 1.5 Auth handler
- [ ] `POST /auth/otp/send` — request: `{phone}`; response: `{ttl_seconds}`
- [ ] `POST /auth/otp/verify` — request: `{phone, code, name}`; response: `{tokens, user}`
- [ ] `POST /auth/refresh` — request: `{refresh_token}`; response: `{tokens}`
- [ ] Валидация входных данных (`validator`)
- [ ] Middleware: `AuthMiddleware` — проверка Bearer, добавление `client_id` в context

### 1.6 Клиентский репозиторий
- [ ] `internal/repository/client_repo.go` — interface
- [ ] `internal/adapter/http/backend_client.go` — реализация (пока заглушка / in-memory для тестов)
- [ ] `GetByPhone`, `Create`, `GetByID`, `Update`, `Delete`

### 1.7 Интеграционные тесты
- [ ] `SendOTP` → `VerifyOTP` → доступ к защищённому endpoint
- [ ] Rate limit: 2 отправки подряд → 429
- [ ] Invalid OTP 3 раза → блокировка
- [ ] Expired OTP → 400

**Критерий приёмки Итерации 1**
```bash
curl -X POST http://localhost:8080/auth/otp/send -d '{"phone":"+79161234567"}'
curl -X POST http://localhost:8080/auth/otp/verify -d '{"phone":"+79161234567","code":"123456","name":"Иван"}'
# ответ: access_token + refresh_token

curl -H "Authorization: Bearer <token>" http://localhost:8080/profile
# 200 / 401 при невалидном токене
```

---

## Итерация 2: Слоты и мастера (read-only + кэш)
> Цель: Защищённые эндпоинты `/slots`, `/slots/{id}`, `/masters` с фильтрами и кэшированием.  
> Длительность: **2–3 дня**

### 2.1 Домен: слоты, программы, мастера
- [ ] `internal/domain/slot.go` — `Slot`, `Program`, `Master` structs; `SlotStatus` enum
- [ ] `internal/domain/slot.go` — `SlotFilter` struct (date_from, date_to, program_type[], master_id[], only_available, limit, offset)

### 2.2 Backend HTTP-клиент (resty)
- [ ] `internal/adapter/http/backend_client.go` — `BackendClient` struct
- [ ] `ListSlots(ctx, filter) → ([]Slot, total, error)`
- [ ] `GetSlot(ctx, id) → (*Slot, error)`
- [ ] `ListMasters(ctx) → ([]Master, error)`
- [ ] Маппинг JSON-ответа бэкенда → domain structs
- [ ] Обработка ошибок бэкенда: 404 → `ErrNotFound`, 500 → wrap error

### 2.3 Кэширование справочников
- [ ] `internal/adapter/redis/slot_cache.go` — `SlotCache` (optional, если кэш в Redis)
- [ ] **Решение:** in-memory кэш (`sync.Map` / `ristretto`) для мастеров и программ (TTL 1 час)
- [ ] Кэш инвалидируется при старте и обновляется фоновым тикером
- [ ] Слоты НЕ кэшируются (часто меняются) или кэш 1 мин

### 2.4 Slot service
- [ ] `internal/service/slot_service.go` — `SlotService`
- [ ] `ListSlots(filter)` — проксирование на бэкенд, применение дефолтов (date_from=now, date_to=+7 дней)
- [ ] `GetSlot(id)` — проксирование
- [ ] `ListMasters()` — проксирование с кэшем

### 2.5 Slot handler
- [ ] `GET /slots` — query params: `date_from`, `date_to`, `program_type`, `master_id`, `only_available`, `limit`, `offset`
- [ ] Парсинг query params, валидация (date_from ≤ date_to, limit ≤ 100)
- [ ] `GET /slots/{slotId}` — path param, UUID validation
- [ ] `GET /masters` — список мастеров (cached)
- [ ] Response DTOs (list + pagination meta)
- [ ] Error handling: `ErrNotFound` → 404, backend timeout → 502

### 2.6 Клиентские заглушки для тестов
- [ ] Wiremock / in-memory backend mock для интеграционных тестов
- [ ] Фикстуры: 3 программы, 4 мастера, 10 слотов (разные статусы)

### 2.7 Тесты
- [ ] Unit: `SlotService` с mock backend client
- [ ] Integration: `GET /slots` → 200 с filtрами
- [ ] Тест фильтров: date range, program_type[], master_id[], only_available

**Критерий приёмки Итерации 2**
```bash
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/slots?date_from=2026-09-01T00:00:00Z&program_type=handbuilding,wheel&limit=10"

curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/slots/$(uuid)

curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/masters
```

---

## Итерация 3: Бронирования (Booking)
> Цель: Полный flow записи и отмены. Доменная валидация + проксирование на бэкенд. Обработка edge cases (нет мест, 410).  
> Длительность: **4–5 дней**

### 3.1 Домен: бронирование
- [ ] `internal/domain/booking.go` — `Booking`, `BookingStatus`, `SeatRequest`
- [ ] `internal/domain/booking.go` — `MaxSeats(freeSeats, capacityCap int) int`
- [ ] `internal/domain/booking.go` — `CanCancel(now, slotStart time.Time) bool` (FR-17: ≥2 часов)
- [ ] `internal/domain/booking.go` — `BookingCreateRequest` struct (`SlotID`, `Seats []SeatRequest`)

### 3.2 Booking repository / backend client
- [ ] `internal/repository/booking_repo.go` — interface: `List(ctx, clientID, limit, offset)`, `GetByID(ctx, clientID, id)`, `Create(ctx, clientID, req)`, `Cancel(ctx, clientID, id)`
- [ ] Реализация в `BackendClient`: POST/GET/DELETE на бэкенд
- [ ] Обработка специфических кодов: 409 → `ErrConflict`, 410 → `ErrGone`

### 3.3 Booking service
- [ ] `internal/service/booking_service.go` — `BookingService`
- [ ] `Create(clientID, req)`:
  - Достать слот (через SlotService / cache)
  - Проверить `status != cancelled` → `ErrSlotCancelled` (410)
  - `len(req.Seats) ≤ MaxSeats(slot.FreeSeats, slot.Program.CapacityCap)` → `ErrSlotFullyBooked` (409)
  - `rentalCount ≤ slot.FreeRentalKits` → `ErrSlotFullyBooked` (409)
  - Проксировать на бэкенд
  - Вернуть `price_total` из ответа (FR-45)
- [ ] `Cancel(clientID, bookingID)`:
  - Достать бронь
  - Достать слот (для `slot.StartAt`)
  - `CanCancel(now, slot.StartAt)` — **справочная проверка** для UX (показать warning), но итоговое решение — бэкенд
  - Проксировать DELETE на бэкенд
  - Маппинг ответа (бэкенд решает: cancelled vs late_cancel)

### 3.4 Booking handler
- [ ] `POST /bookings` — request: `{slot_id, seats:[{rental:true/false}]}`
  - response: `201` с полным объектом `Booking`
  - `409` — мест нет / прокат исчерпан
  - `410` — слот отменён мастерской
- [ ] `GET /bookings` — query: `limit`, `offset`
  - response: list + pagination
- [ ] `GET /bookings/{bookingId}` — 200 / 404
- [ ] `DELETE /bookings/{bookingId}` — 200 / 403 (не ваша бронь) / 404

### 3.5 Error mapping
- [ ] `internal/handler/errors.go` — централизованный маппинг domain error → HTTP code
  - `ErrNotFound` → 404
  - `ErrUnauthorized` → 401
  - `ErrConflict` → 409 (нет мест)
  - `ErrGone` → 410 (слот отменён)
  - `ErrTooManyRequests` → 429
  - `ErrInternal` → 500

### 3.6 Тесты
- [ ] Unit: `BookingService.Create` — pre-check валидация (maxSeats, rental, cancelled)
- [ ] Integration: полный flow создания брони
- [ ] Integration: попытка записи на `cancelled` слот → 410
- [ ] Integration: отмена брони → 200

**Критерий приёмки Итерации 3**
```bash
# Создание брони
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -d '{"slot_id":"...","seats":[{"rental":false},{"rental":true}]}' \
  http://localhost:8080/bookings
# 201 + Booking с price_total

# Повторная запись на тот же слот (если мест нет)
# 409

# Отмена
curl -X DELETE -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/bookings/$(uuid)
# 200
```

---

## Итерация 4: Профиль и Push-токены
> Цель: CRUD профиля, удаление аккаунта, регистрация push-токенов.  
> Длительность: **2 дня**

### 4.1 Профиль
- [ ] `GET /profile` — текущий клиент из JWT context
- [ ] `PATCH /profile` — обновление имени (phone read-only)
- [ ] `DELETE /profile` — удаление аккаунта (FR-48):
  - Активные брони аннулируются (проксирование на бэкенд для освобождения мест)
  - Прошедшие брони анонимизируются (бэкенд обрабатывает)
  - Инвалидация всех refresh tokens
- [ ] Handler + Service + Repository

### 4.2 Push-токены
- [ ] `POST /push/register` — request: `{token, platform}` (ios/android)
- [ ] `DELETE /push/unregister` — удаление токена
- [ ] Хранение в PostgreSQL (`push_tokens` table) или Redis
- [ ] Проксирование push-токена на существующий push-сервис (FCM/APNS adapter — заглушка для MVP)

### 4.3 Схема БД (если PostgreSQL для локальных данных)
- [ ] Применить миграцию `001_init.up.sql` (clients, push_tokens, otp_codes mirror)
- [ ] `golang-migrate` + `docker-compose` для dev
- [ ] sqlc (опционально): генерация типобезопасных запросов

### 4.4 Тесты
- [ ] `PATCH /profile` → обновление имени
- [ ] `DELETE /profile` → 401 на старых токенах, брони аннулированы
- [ ] `POST /push/register` → токен сохранён

**Критерий приёмки Итерации 4**
```bash
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/profile
# {id, name, phone}

curl -X PATCH -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"Новое Имя"}' http://localhost:8080/profile
# 200

curl -X POST -H "Authorization: Bearer $TOKEN" \
  -d '{"token":"fcm-token-123","platform":"android"}' \
  http://localhost:8080/push/register
# 201
```

---

## Итерация 5: Quality of Life — тесты, документация, мониторинг
> Цель: Production-ready уровень: покрытие тестами >70%, Swagger, метрики, graceful degradation.  
> Длительность: **3 дня**

### 5.1 Тесты
- [ ] Unit-покрытие >70% (`go test -cover`)
  - Domain: `MaxSeats`, `CanCancel`, статусы
  - Service: `AuthService`, `SlotService`, `BookingService` (с моками)
  - Handler: HTTP тесты через `httptest` + `chi`
- [ ] Интеграционные тесты:
  - `TestEndToEndAuthFlow` (register → login → refresh → logout)
  - `TestBookingFlow` (slots → book → cancel → profile)
- [ ] `mockery` / ручные моки для интерфейсов repository

### 5.2 API Documentation
- [ ] `swaggo/swag` — `go get github.com/swaggo/http-swagger`
- [ ] Аннотации в handlers (`@Summary`, `@Param`, `@Success`, `@Failure`)
- [ ] `make swagger` → генерация `docs/`
- [ ] Endpoint `/swagger/index.html` (только dev/staging)

### 5.3 Мониторинг (NFR-18, NFR-19)
- [ ] Middleware: Prometheus metrics (`http_requests_total`, `http_request_duration_seconds`)
- [ ] Бизнес-метрики: `bookings_created_total`, `bookings_cancelled_total`, `otp_sent_total`
- [ ] Health check: `/health` (+ readiness: проверка Redis, Backend)
- [ ] Structured logging: request_id, client_id передаются в zap fields
- [ ] Error tracking: 5xx логируются с trace-id и stack trace

### 5.4 Resilience
- [ ] Circuit breaker для backend client (опционально: `sony/gobreaker`)
- [ ] Retry policy для idempotent запросов (GET, DELETE) — resty встроенный retry
- [ ] Timeout: BackendTimeout 5s (NFR-1/2)
- [ ] Graceful degradation: если backend недоступен → 502 с понятным сообщением

### 5.5 Deployment
- [ ] `Dockerfile` оптимизирован (multi-stage, distroless/alpine)
- [ ] `docker-compose.yml` с PostgreSQL + Redis + app
- [ ] `.env.production.example` с required переменными
- [ ] Helm chart / k8s manifests (опционально)

### 5.6 Документация проекта
- [ ] `README.md` — запуск, структура, переменные окружения
- [ ] `API.md` — краткое описание эндпоинтов (или ссылка на Swagger)
- [ ] `ARCHITECTURE.md` — C4 diagram / описание слоёв

**Критерий приёмки Итерации 5**
```bash
make test        # покрытие >70%
make lint        # 0 ошибок
make swagger     # docs/swagger.json создан
make build       # bin/api собран
docker compose up # приложение + redis + postgres поднимаются

curl http://localhost:8080/metrics
# prometheus metrics

curl http://localhost:8080/health
# {"status":"ok","redis":"ok","backend":"ok"}
```

---

## Итоговая дорожная карта

| Итерация | Длительность | Что получается |
|----------|-------------|----------------|
| **0. Инфраструктура** | 1–2 дня | Рабочий сервер, конфиг, Docker, логи |
| **1. Auth / OTP / JWT** | 3–4 дня | Регистрация, вход, сессии, rate limiting |
| **2. Слоты + Мастера** | 2–3 дня | Просмотр расписания с фильтрами и кэшем |
| **3. Бронирования** | 4–5 дня | Запись, отмена, валидация, обработка ошибок |
| **4. Профиль + Push** | 2 дня | Профиль, удаление аккаунта, уведомления |
| **5. Quality** | 3 дня | Тесты >70%, Swagger, метрики, документация |
| **ИТОГО** | **15–19 дней** (~3–4 недели) | MVP back-end готов |

---

## Зависимости между итерациями (DAG)

```
Iter 0 (скелет)
    │
    ├──► Iter 1 (Auth) ──► зависит от Redis, JWT, конфига
    │       │
    │       └──► Iter 2 (Слоты) ──► зависит от Auth middleware
    │               │
    │               └──► Iter 3 (Бронирования) ──► зависит от Слоты + Auth
    │                       │
    │                       └──► Iter 4 (Профиль) ──► зависит от Auth + Брони
    │                               │
    └───────────────────────────────┴──► Iter 5 (Quality) ──► зависит от всего
```

---

## Чеклист готовности к продакшену

### Безопасность
- [ ] `JWT_SECRET` ≥ 32 байт, хранится в vault
- [ ] `REDIS_PASS` включён в production
- [ ] `Environment=production` отключает Swagger, debug endpoints
- [ ] TLS 1.2+ на ingress (NFR-6)
- [ ] CORS настроен только для домена приложения

### Производительность
- [ ] NFR-1: p95 /slots ≤ 500 мс (нагрузочное тестирование)
- [ ] NFR-2: p95 POST /bookings ≤ 800 мс
- [ ] Redis кэш для справочников включён
- [ ] Connection pool к бэкенду настроен (resty)

### Надёжность
- [ ] Graceful shutdown: drain in-flight requests (10s timeout)
- [ ] Health/readiness probes настроены
- [ ] Логи пишутся в stdout (12-factor)
- [ ] Ротация логов: 30 дней (NFR-18)

### Документация
- [ ] Swagger доступен на staging
- [ ] Postman коллекция экспортирована (опционально)
- [ ] Runbook: что делать при падении backend / Redis