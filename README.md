# TripGo Service

Сервис поездок TripGo: создание, получение и завершение поездки. Лабораторная работа 2 курса «Разработка микросервисов на Go» (Архитектура, качество кода и observability).

## Требования

- Go 1.24+ (инструменты подключены через `go get -tool`)
- Docker (для локального окружения)
- `tripgoctl` — утилита курса, см. [course-infra](https://github.com/course-go-autumn-2026/course-infra)
- `psql` — опционально, для ручной проверки базы

## Запуск

```bash
tripgoctl environment start    # поднимает PostgreSQL и коллектор телеметрии
make migrate                   # накатывает миграции
make run                       # запускает сервис
```

Сервис слушает `HTTP_ADDR` (по умолчанию `:8080`).

Проверка:
```bash
curl -i localhost:8080/health
curl -i localhost:8080/ready
```

## Цели Makefile

| Цель | Что делает |
|---|---|
| `make run` | запускает сервис |
| `make generate` | генерирует типы, chi-сервер и моки (`mockgen`) |
| `make migrate` | накатывает все миграции (goose up) |
| `make migrate-down` | откатывает последнюю миграцию (goose down) |
| `make test` | запускает unit-тесты с флагом `-race` и печатает покрытие |
| `make test-integration` | запускает интеграционные тесты с реальной БД |
| `make lint` | запускает `golangci-lint` с проверкой слоёв |

## Переменные окружения

Все переменные обязательны: если какая-то не задана или длительность не больше нуля, сервис падает на старте с понятной ошибкой.

| Переменная | Пример | Назначение |
|---|---|---|
| `HTTP_ADDR` | `:8080` | адрес HTTP-сервера |
| `HTTP_READ_HEADER_TIMEOUT` | `5s` | время на чтение заголовков |
| `HTTP_READ_TIMEOUT` | `10s` | время на чтение всего запроса |
| `HTTP_WRITE_TIMEOUT` | `15s` | время на обработку и запись ответа |
| `HTTP_IDLE_TIMEOUT` | `60s` | время жизни keep-alive соединения |
| `LOG_LEVEL` | `info` | уровень логирования |
| `SHUTDOWN_TIMEOUT` | `10s` | бюджет на graceful shutdown |
| `DATABASE_URL` | из `.env` | строка подключения к PostgreSQL |
| `DATABASE_MAX_CONNS` | `10` | максимум соединений в пуле |
| `DATABASE_MIN_CONNS` | `2` | минимум соединений в пуле |
| `DATABASE_MAX_CONN_LIFETIME` | `30m` | время жизни соединения |
| `DATABASE_CONNECT_TIMEOUT` | `5s` | таймаут на подключение к БД |
| `DATABASE_QUERY_TIMEOUT` | `3s` | таймаут на каждый запрос в БД |

## API

| Метод | Путь | Ответы |
|---|---|---|
| `POST` | `/api/v1/trips` | 201, 400, 409, 500 |
| `GET` | `/api/v1/trips/{tripId}` | 200, 400, 404, 500 |
| `POST` | `/api/v1/trips/{tripId}/finish` | 200, 400, 404, 409, 500 |
| `GET` | `/health` | 200 |
| `GET` | `/ready` | 200, 503 |

Ошибки отдаются по RFC 9457 (`application/problem+json`) с полем `code`: `invalid_request`, `trip_not_found`, `trip_completed`, `driver_busy`, `internal_error`.

## Структура

```text
cmd/service/                  main: конфиг, логгер, сигналы; всё остальное делает container
internal/
  config/                     разбор и проверка переменных окружения
  container/                  создаёт все зависимости и связывает их (единственное место сборки)
  domain/trip/                модель поездки, переходы статусов, доменные ошибки
  usecase/trip/               create/, get/, finish/: сценарии, у каждого свой contract.go и моки
  http/                       транспорт
    trip/create|get|finish/   хендлеры по ручкам
    trip/dto/                 доменная поездка → ответ по контракту
    domainerror/              доменная ошибка → problem+json, единственное место
    middleware/               Recover, Tracing, Logging, Metrics, проверка по контракту
    router/                   сборка middleware и ручек в один роутер
    health/, response/, server/
  repository/trip/            pgx + squirrel
  infra/
    postgres/                 пул, transaction/ — менеджер транзакций
    telemetry/                логгер, провайдеры OpenTelemetry, метрики поездок
  generated/                  код по OpenAPI (руками не правится)
migrations/
```

## Архитектура

Слои с однонаправленными зависимостями. Зависимости направлены внутрь: транспорт и репозитории знают про usecase, usecase знает про домен, домен не знает ни про кого.

```text
main → container → {http, usecase, repository, infra}
                ↓
             usecase → domain
```

- `domain/trip` не импортирует `net/http`, `chi`, `pgx`, `squirrel`, `otel`, `api`.
- `usecase/trip/*` не импортирует транспорт, драйвер БД, реализацию хранилища, инфраструктуру.
- `http/*` не импортирует `repository/*` и `infra/*`.

Правила проверяются `depguard` в `.golangci.yml` — договорённость стала частью сборки.

### Интерфейсы

Объявлены потребителями, в `contract.go` каждого пакета:
- `usecase/trip/create/contract.go` — `tripRepository` (только `Save`), `historyRepository` (только `Add`), `txManager`, `metrics` (только `TripCreated`).
- `usecase/trip/finish/contract.go` — `tripRepository` (только `GetForUpdate`, `Update`), `historyRepository`, `txManager`, `metrics`.
- `http/trip/*/contract.go` — `useCase` с одним методом `Execute`.
- `http/health/contract.go` — `pinger` с одним методом `Ping`.

Моки — `mockgen` через `//go:generate`, лежат в `mocks_test.go` рядом с `contract.go`. Отдельного `mocks/` нет.

### Сборка

`main` читает конфиг, зовёт `container.New(cfg)`, запускает сервер, ждёт SIGTERM. Всё создание реализаций — в `container`. Больше нигде нет `pgxpool.New`, `otelpgx.NewTracer`, конструкторов репозиториев.

### Решения

- **`domain.Trip` вместо отдельного `model.Trip`** — одна сущность, копия была бы дублированием инвариантов. Если появятся DTO для внешнего мира — заведём `model/` точечно.
- **Свой `Tracing` вместо `otelhttp`** — `otelhttp.NewHandler` не переопределяет имя спана на шаблон маршрута без хаков. Используем тот же `routeResolver`, что и `Logging`/`Metrics`.
- **`get` без бизнес-метрик** — по ТЗ п. 6.3 считаем только `created` и `completed`. Чтение покрывается `http.server.request.duration` и спанами Jaeger. Счётчик `read` — это шум.

## Observability

### Трейсы

`Tracing` middleware создаёт корневой спан `{method} {route_template}` — например, `POST /api/v1/trips/{tripId}/finish`, а не полный URL. `otelpgx` инструментирует пул: каждый запрос в БД порождает вложенный спан с `db.operation.name`, `db.namespace` (новая семантика OTel).

### Метрики

| Метрика | Тип | Единица | Источник |
|---|---|---|---|
| `http.server.request.duration` | histogram | s | middleware `Metrics` |
| `http.server.active_requests` | up-down counter | {request} | middleware `Metrics` |
| `db.client.operation.duration` | histogram | s | `otelpgx` |
| `tripgo.trips.created` | counter | {trip} | `usecase/trip/create` |
| `tripgo.trips.completed` | counter | {trip} | `usecase/trip/finish` |

### Логи

`slog` + JSON + stdout, уровень из `LOG_LEVEL`. Кастомный `traceHandler` добавляет `trace_id`/`span_id` из контекста. Middleware `Logging` пишет одну запись на HTTP-запрос: `method`, `route` (шаблон), `status`, `duration_ms`, `trace_id`, `span_id`. Уровень `Info` для 2xx/4xx, `Error` для 5xx. Тела запроса/ответа не логируем.

Бизнес-события (`trip created`, `trip completed`) пишутся в usecase.

### Порядок middleware

`Recover → Tracing → Logging → Metrics → validate → router`.

## Решения по бизнес-логике

### Уровень изоляции: READ COMMITTED

Транзакция в сервисе нужна ради атомарности нескольких записей (поездка и запись в журнале статусов), а не ради согласованного чтения многих строк. От гонок защищают ограничения базы, а не изоляция:
- две активные поездки одного водителя запрещены уникальным индексом;
- двойное завершение исключено условием `status = 'active'` прямо в `UPDATE`.

Уровень задаётся в одном месте, в `transaction.Manager.Do`.

### Менеджер транзакций

`transaction.Manager.Do(ctx, fn)` открывает транзакцию, кладёт её в `context` и вызывает `fn` с этим контекстом. Если `fn` вернула `nil`, делается `COMMIT`; если ошибку или случилась паника — `ROLLBACK`. Откат выполняется с `context.WithoutCancel` и с собственным таймаутом, чтобы зависшее соединение не держало откат бесконечно.

Репозитории не получают транзакцию аргументом. Они берут исполнителя из контекста через `transaction.ExecutorFromContext`.

### Запрет двух активных поездок на водителя

Частичный уникальный индекс:
```sql
CREATE UNIQUE INDEX trips_unique_driver_id_idx ON trips (driver_id) WHERE status = 'active';
```
Обычный `UNIQUE (driver_id)` не подходит: он запретил бы водителю иметь больше одной поездки вообще. Из двух одновременных `INSERT` один получает ошибку `23505`. Репозиторий распознаёт её по коду и имени индекса и возвращает доменную ошибку `ErrDriverBusy`, которая превращается в `409 driver_busy`.

### Завершение поездки без гонки

В `usecase/finish` чтение выполняется с `FOR UPDATE`. Если два одновременных `finish` придут вместе, первый заблокирует строку, завершит поездку, второй дождётся снятия блокировки, увидит статус `completed` и вернёт `ErrTripCompleted` (409).

## Проверки

- `goose down` откатывает обе миграции;
- `/ready` отдаёт `503` при остановленном PostgreSQL и `200` после старта;
- по `SIGTERM` сервис дожидается текущего запроса и выходит в пределах `SHUTDOWN_TIMEOUT`;
- при падении второй вставки в `trips` ничего не остаётся (проверено интеграционным тестом);
- при недоступности OTel-коллектора сервис продолжает отвечать (в логах WARN от `otel.SetErrorHandler`).