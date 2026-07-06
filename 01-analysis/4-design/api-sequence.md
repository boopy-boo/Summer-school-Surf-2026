# Диаграммы последовательности API

> Этап 3. Ключевые сценарии взаимодействия клиентского приложения с API.

## UC-1. Запись на занятие

```mermaid
sequenceDiagram
    actor C as Клиент
    participant A as Приложение
    participant API as API /slots, /bookings
    participant B as Бэкенд

    C->>A: Выбирает слот, нажимает «Записаться»
    A->>API: GET /slots/{slotId}
    API-->>A: Slot (свободно мест, цена, прокат)
    A->>A: Показывает экран записи (SCR-004)

    C->>A: Выбирает число мест и инструменты
    A->>API: POST /bookings
    Note over A,API: {slot_id, seats:[{rental:false}, {rental:true}]}

    API->>B: Атомарная проверка мест и проката
    alt Успех
        B-->>API: Booking (id, price_total, status=active)
        API-->>A: 201 Created + Booking
        A->>A: Показывает BS-002 (подтверждение)
        Note over A: После первой записи:<br/>запрос на push-разрешение
    else Нет мест
        B-->>API: Отказ (конфликт)
        API-->>A: 409 Conflict
        A->>A: Ошибка «Мест нет» + предложение выбрать другой слот
    else Слот отменён
        B-->>API: Отказ (слот недоступен)
        API-->>A: 410 Gone
        A->>A: «Занятие отменено мастерской»
    end
```

## UC-2. Отмена записи

```mermaid
sequenceDiagram
    actor C as Клиент
    participant A as Приложение
    participant API as API /bookings
    participant B as Бэкенд

    C->>A: Открывает «Мои бронирования» (SCR-005)
    A->>API: GET /bookings?limit=20&offset=0
    API-->>A: Список броней

    C->>A: Выбирает бронь, нажимает «Отменить»
    A->>A: Показывает BS-003 (подтверждение)
    C->>A: Подтверждает отмену
    A->>API: DELETE /bookings/{bookingId}

    API->>B: Определяет тип отмены по времени
    alt Ранняя отмена (≥2ч)
        B-->>API: Booking (status=cancelled)
        API-->>A: 200 OK + Booking
        A->>A: Снек «Бронь отменена», обновляет список
        Note over B: Места и прокат возвращены в слот
    else Поздняя отмена (<2ч)
        B-->>API: Booking (status=late_cancel)
        API-->>A: 200 OK + Booking
        A->>A: Снек «Поздняя отмена: место не освобождено»
        Note over B: Место и прокат НЕ освобождены
    end
```

## UC-3. Отмена слота мастерской → push клиенту

```mermaid
sequenceDiagram
    participant Admin as Админка (вне скоупа)
    participant B as Бэкенд
    participant PNS as Push-сервис
    participant A as Приложение
    actor C as Клиент

    Admin->>B: Отменяет слот (причина: «Сломалась печь»)
    B->>B: Slot.status = cancelled
    B->>B: Связанные Booking.status = workshop_cancelled
    B->>PNS: Отправить push записанным клиентам

    PNS-->>A: Push: «Занятие отменено: Сломалась печь»
    A->>A: Показывает уведомление
    C->>A: Тап на push
    A->>A: Открывает SCR-006 (детали брони)
    Note over A: Статус: «Отменено мастерской»<br/>Причина: «Сломалась печь»<br/>CTA «Записаться» неактивен
```

## UC-4. Фильтрация слотов

```mermaid
sequenceDiagram
    actor C as Клиент
    participant A as Приложение
    participant API as API /slots

    C->>A: Открывает «Занятия» (SCR-002)
    A->>API: GET /slots (дефолт: сегодня + 7 дней)
    API-->>A: SlotListResponse

    C->>A: Нажимает «Фильтры»
    A->>A: Открывает BS-001
    C->>A: Выбирает программу, дату, мастера
    A->>API: GET /slots?program_type=handbuilding&date_from=...&master_id=...
    API-->>A: Отфильтрованный список
    A->>A: Обновляет SCR-002 с индикатором активных фильтров
```

## UC-5. Вход / Регистрация (SMS OTP)

```mermaid
sequenceDiagram
    actor C as Клиент
    participant A as Приложение
    participant API as API /auth

    C->>A: Вводит имя и телефон
    A->>API: POST /auth/otp/send {phone:+7916...}
    API-->>A: 204 No Content (SMS отправлено)

    C->>A: Вводит 6-значный код
    A->>API: POST /auth/otp/verify {phone, code, name}
    alt Код верный
        API-->>A: 200 OK {tokens, user}
        A->>A: Сохраняет JWT
        A->>A: Переход на SCR-002 (Занятия)
    else Код неверный / истёк
        API-->>A: 401 Unauthorized
        A->>A: «Неверный или истёкший код»
    else Слишком часто
        API-->>A: 429 Too Many Requests
        A->>A: «Попробуйте через 60 секунд»
    end