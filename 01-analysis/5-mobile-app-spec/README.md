# ТЗ на мобильное приложение «Глина»

> Этап 5. Детальное техническое задание на клиентское мобильное приложение «Глина»
> (самостоятельная запись на мастер-классы, роль «Клиент»).

**Статус:** Актуален · **Версия:** 1.0 · **Дата:** 2026-07-01

## Экраны и шторки

| ID | Экран / Шторка | Тип | Зона | Приоритет | ТЗ |
|----|----------------|-----|------|-----------|-----|
| SCR-001 | Регистрация / Вход | Экран | НЗ | Critical | [SCR-001-registration.md](SCR-001-registration.md) |
| SCR-002 | Список слотов | Экран | АЗ | Critical | [SCR-002-slot-list.md](SCR-002-slot-list.md) |
| BS-001 | Фильтры | Bottom Sheet | АЗ | High | [BS-001-filters.md](BS-001-filters.md) |
| SCR-003 | Карточка слота | Экран | АЗ | Critical | [SCR-003-slot-card.md](SCR-003-slot-card.md) |
| SCR-004 | Оформление записи | Экран | АЗ | Critical | [SCR-004-booking.md](SCR-004-booking.md) |
| BS-002 | Подтверждение записи | Bottom Sheet | АЗ | High | [BS-002-booking-success.md](BS-002-booking-success.md) |
| SCR-005 | Мои бронирования | Экран | АЗ | Critical | [SCR-005-my-bookings.md](SCR-005-my-bookings.md) |
| SCR-006 | Детали брони + отмена | Экран | АЗ | Critical | [SCR-006-booking-details.md](SCR-006-booking-details.md) |
| BS-003 | Подтверждение отмены | Bottom Sheet | АЗ | High | [BS-003-cancel-confirm.md](BS-003-cancel-confirm.md) |
| SCR-007 | Профиль клиента | Экран | АЗ | Medium | [SCR-007-profile.md](SCR-007-profile.md) |

> **Зоны:** НЗ — неавторизованная, АЗ — авторизованная.

## API

Все запросы REST, спецификации — [`../api/`](../api/) (домены `auth`, `profile`, `slots`, `bookings`, `masters`).

## Соглашения

- **Платформа:** нативное мобильное приложение (iOS + Android)
- **Числа не хардкодятся:** потолки программ, прокатный фонд, цены приходят из API