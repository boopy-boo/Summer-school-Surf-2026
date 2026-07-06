# Гончарная мастерская «Глина» — Клиентское приложение и API

> Полный цикл: аналитика → разработка Go-бэкенда → тестовая документация.
> Проектная работа: мобильное приложение для записи клиентов на мастер-классы
> с API на Go (Chi), PostgreSQL, Redis, JWT-авторизацией по SMS OTP.

---

## Содержание

- [О проекте](#о-проекте)
- [Структура репозитория](#структура-репозитория)
- [Архитектура](#архитектура)
- [API Endpoints](#api-endpoints)
- [Скоуп и границы](#скоуп-и-границы)
- [Ключевые решения](#ключевые-решения)
- [Как запустить](#как-запустить)
- [Стек технологий](#стек-технологий)

---

## О проекте

**Заказчица:** Марина, владелица гончарной мастерской «Глина».

**Проблема:** Запись на занятия велась вручную через Instagram-директ и ежедневник — двойные брони, путаница, потерянные клиенты.

**Решение:** Клиентское мобильное приложение, в котором пользователи самостоятельно:
- просматривают расписание на 7 дней вперёд,
- фильтруют по программе, мастеру, дате,
- записываются на занятие (1–3 места, свои или прокатные инструменты),
- управляют бронями (отмена с учётом 2-часового окна),
- получают push-напоминания.

Вся инфраструктура мастерской (расписание, мастера, программы, админка) — в существующем black-box бэкенде. Клиентское приложение их потребляет через API.

---

## Структура репозитория

```
pottery-workshop/
├── README.md                     # ← вы здесь
│
├── 01-analysis/                  # Аналитика: от брифа до OpenAPI
│   ├── 0-customer-brief/         # Исходное письмо заказчицы (сырой бриф)
│   ├── 1-elicitation/            # Уточняющие вопросы, ответы, описание домена
│   ├── 2-requirements/           # FR, NFR, use cases, user stories
│   ├── 3-design-brief/           # Дизайн-бриф, foundations, экранные спеки
│   ├── 4-design/                 # ERD, sequence-диаграммы, навигация
│   ├── 5-mobile-app-spec/        # Фича-лист с приоритетами и трассировкой
│   ├── api/                      # OpenAPI 3.0.3 (auth, slots, bookings, profile, masters, common)
│   ├── checklists/               # Чеклисты качества документации
│   └── prompts/                  # Примеры хороших и плохих промптов для ИИ
│
├── 02-development/               # Разработка: Go-бэкенд клиентского API
│   ├── cmd/api/main.go           # Точка входа
│   ├── internal/
│   │   ├── config/               # Конфигурация (env)
│   │   ├── domain/               # Доменные сущности и чистая логика (Client, Slot, Booking, Program, Master)
│   │   ├── handler/              # HTTP-хендлеры (auth, slots, bookings, middleware)
│   │   ├── service/              # Бизнес-логика (auth, slot, booking)
│   │   ├── adapter/              # Адаптеры: postgres (репозиторий), redis (OTP), http (backend client)
│   │   └── http/                 # Сервер (Chi)
│   ├── migrations/               # SQL-миграции PostgreSQL (001_init.up.sql / down.sql)
│   ├── ui-prototype/             # HTML-прототип ключевых экранов
│   ├── Dockerfile                # Контейнеризация API
│   ├── docker-compose.yml        # Postgres + Redis + API
│   ├── go.mod / Makefile         # Модули и команды сборки
│   └── *.md                      # Документы: DOMAIN-ENTITIES, GO-ARCHITECTURE, SCOPE-ANALYSIS, IMPLEMENTATION-PLAN
│
└── 03-tests/                     # Тестовая документация
    └── TEST-CASES.md             # 79+ тест-кейсов с трассировкой FR/NFR → TC
```

---

## Архитектура

![Clean Architecture / Ports-and-Adapters](02-development/GO-ARCHITECTURE.md)

```
┌─────────────────────────────────────────┐
│  HTTP Layer (Chi router)                │
│  handler/  →  middleware (JWT auth)     │
├─────────────────────────────────────────┤
│  Service Layer                          │
│  service/auth_service.go                │
│  service/slot_service.go                │
│  service/booking_service.go             │
├─────────────────────────────────────────┤
│  Domain Layer (чистые структуры)        │
│  domain/client.go, slot.go, booking.go  │
│  domain/errors.go                       │
├─────────────────────────────────────────┤
│  Adapter Layer                          │
│  adapter/postgres/  → ClientStore       │
│  adapter/redis/     → OTPStore          │
│  adapter/http/      → BackendClient     │
└─────────────────────────────────────────┘
```

- **JWT:** access = 24 ч, refresh = 30 дней (NFR-7)
- **OTP:** 6 цифр, TTL = 5 мин, rate limit = 1/60 сек, max попыток = 3 (NFR-10)
- **Брутфорс-защита:** 5 неудач/час → блок 30 мин (NFR-11)

---

## API Endpoints

| Метод | Путь | Описание | Auth |
|:---|:---|:---|:---|
| `POST` | `/auth/otp/send` | Запрос SMS OTP | — |
| `POST` | `/auth/otp/verify` | Проверка OTP, вход/регистрация | — |
| `POST` | `/auth/refresh` | Обновление access-токена | — |
| `GET` | `/slots` | Список слотов с фильтрами | Bearer |
| `GET` | `/slots/{slotId}` | Карточка слота | Bearer |
| `GET` | `/masters` | Справочник мастеров | Bearer |
| `GET` | `/bookings` | Мои бронирования (пагинация) | Bearer |
| `POST` | `/bookings` | Запись на занятие | Bearer |
| `GET` | `/bookings/{bookingId}` | Детали брони | Bearer |
| `DELETE` | `/bookings/{bookingId}` | Отмена брони | Bearer |
| `GET` | `/profile` | Профиль клиента | Bearer |
| `PATCH` | `/profile` | Обновление профиля | Bearer |
| `DELETE` | `/profile` | Удаление аккаунта | Bearer |

Полная OpenAPI-спецификация — в [`01-analysis/api/`](01-analysis/api/).

---

## Скоуп и границы

| В скоупе | Вне скоупа (существующая инфраструктура / Phase 2) |
|:---|:---|
| Клиентское приложение + API для него | Интерфейсы мастера и администратора |
| Чтение слотов, программ, мастеров (read-only) | CRUD расписания и программ |
| Бронирование, отмена, мои брони | Онлайн-оплата (пока наличные/перевод) |
| Профиль клиента (имя, телефон, удаление) | Оценки мастеров (Phase 2) |
| Push-напоминания и уведомления | Программа лояльности (Phase 2) |
| | «Поделиться» занятием (Phase 2) |

---

## Ключевые решения

| Решение | Основание |
|:---|:---|
| SMS OTP для авторизации | FR-43, NFR-10 |
| Push-уведомления (системные) | FR-33, NFR-8 |
| Оффлайн-оплата (наличные/перевод) | FR-30, ограничения на старте |
| Граница отмены = 2 часа до начала | FR-17, FR-18 |
| Макс. 3 места на бронь | FR-12, FR-13 |
| Программы: лепка ≤6, круг ≤10 мест | FR-13, доменные ограничения |
| Оценки мастеров перенесены в Phase 2 | Согласовано с заказчицей 2026-07-01 |
| «Поделиться» — декоративная иконка в MVP | Согласовано с заказчицей 2026-07-01 |
| Время в UTC, отображение в локальной зоне клиента | NFR-17 |

---

## Как запустить

```bash
cd 02-development

# Docker: Postgres + Redis + API
docker-compose up --build

# Локально (требуется Postgres + Redis)
make migrate-up
go run cmd/api/main.go
```

Переменные окружения см. в [`02-development/internal/config/config.go`](02-development/internal/config/config.go).

---

## Стек технологий

| Слой | Технология |
|:---|:---|
| Язык | Go 1.22+ |
| HTTP-роутер | Chi (go-chi/chi/v5) |
| База данных | PostgreSQL 16 |
| Кэш / OTP | Redis 7 |
| JWT | github.com/golang-jwt/jwt/v5 |
| Валидация | github.com/go-playground/validator/v10 |
| Миграции | golang-migrate/migrate |
| Контейнеры | Docker, Docker Compose |

---

## Связи между артефактами

| Аналитика | Разработка | Тесты |
|:---|:---|:---|
| `01-analysis/2-requirements/functional-requirements.md` (FR-13) | `02-development/internal/domain/booking.go` → `MaxSeats()` | `03-tests/TEST-CASES.md` → TC-UNIT-001..003 |
| `01-analysis/4-design/data-model.md` | `02-development/internal/domain/*.go` → структуры | `03-tests/TEST-CASES.md` → все TC с тегом `domain` |
| `01-analysis/api/bookings/api.yaml` | `02-development/internal/handler/booking_handler.go` | `03-tests/TEST-CASES.md` → TC-BOOK-xxx |
| `01-analysis/2-requirements/non-functional-requirements.md` | `02-development/internal/service/auth_service.go` → rate limits | `03-tests/TEST-CASES.md` → TC-NFR-xxx, TC-AUTH-002..004 |

---"# Summer-school-Surf-2026" 
