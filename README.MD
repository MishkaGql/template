# Trip Service

Go-сервис создания, получения и завершения поездок. PostgreSQL хранит поездки и историю статусов. HTTP-контракт находится в `contracts/openapi/trip-service.openapi.yaml`, правила разработки — в `docs/`.

## Требования

- Go 1.27.2 или новее.
- `make` и `tripgoctl` для локальной инфраструктуры.
- Docker для интеграционных тестов.

Инструменты генерации, миграций и моков закреплены в `go.mod` и запускаются через `go tool`.

## Локальный запуск

В корне репозитория выполните:

```sh
tripgoctl cluster start
tripgoctl environment start
tripgoctl connect
make migrate
make run
```

`tripgoctl` записывает адреса PostgreSQL, Push Service и коллектора в `.env`. При использовании собственного окружения скопируйте `.env.example` в `.env` и укажите свои адреса. `make generate` обновляет HTTP-сервер, клиент Push Service и тестовые моки.

## Проверка сервиса

```sh
curl http://localhost:8080/health
curl http://localhost:8080/ready
```

Оба эндпоинта возвращают `200` и `{"status":"ok"}`, когда сервис и БД доступны. При недоступной БД `/ready` возвращает `503` и `{"status":"unavailable"}`.

Создание поездки:

```sh
curl -i http://localhost:8080/api/v1/trips \
  -H 'Content-Type: application/json' \
  -d '{
    "user_id":"5cb72c04-7650-45c9-a79b-bcdba0631e0c",
    "driver_id":"8860b315-ec86-42eb-a17c-7c163d721ff5",
    "price":1450,
    "start_point":{"latitude":59.9398,"longitude":30.3146},
    "end_point":{"latitude":59.929,"longitude":30.3626}
  }'
```

Ответ `201` содержит поездку и заголовок `Location`. По возвращённому `id` доступны `GET /api/v1/trips/{id}` и `POST /api/v1/trips/{id}/finish`.

## Опрос устаревших поездок

`сканер → ограниченная FIFO-очередь → воркеры → Push Service`

Каждый тик сканер одним запросом выбирает активные поездки, начиная с самой давно молчащей. Возраст отсчитывается от последней координаты или начала поездки. Если очередь заполнена, сканер пропускает задачу, пишет Warn и увеличивает `tripgo.stale_checks{result="dropped"}`. Поездка снова участвует в следующем опросе. Пока задача ожидает или выполняется, повторная постановка исключена.

| Переменная | По умолчанию | Назначение |
|---|---|---|
| `POSITION_CHECK_INTERVAL` | `30s` | Период сканирования и повторного запроса координаты |
| `POSITION_STALE_AFTER` | `60s` | Возраст координаты, после которого нужен опрос |
| `POSITION_WORKERS` | `4` | Максимум одновременных запросов |
| `POSITION_QUEUE_SIZE` | `100` | Число ожидающих задач |
| `PUSH_HTTP_URL` | Из `.env` от tripgoctl | Базовый адрес Push Service |
| `PUSH_REQUEST_TIMEOUT` | `1s` | Таймаут одной попытки |
| `SHUTDOWN_TIMEOUT` | `10s` в окружении курса | Общий бюджет остановки |

`POST /api/v1/trips/{tripId}/positions` сохраняет точку. PostgreSQL атомарно обновляет `last_position_at` до максимального времени записи. После успешного коммита ожидающий пуш отменяется. Уже начатый HTTP-вызов имеет собственный таймаут. `GET /api/v1/trips/{tripId}/positions` возвращает маршрут по возрастанию `recorded_at`, затем `id`.

При SIGTERM сервер завершает HTTP-запросы, сканер закрывает очередь, воркеры дорабатывают задачи. По истечении общего бюджета исходящие запросы отменяются. После рестарта сканер восстанавливает список кандидатов из PostgreSQL.

Метрики очереди и занятых воркеров — `tripgo.position_requests.queue.size` и `tripgo.position_requests.workers.active`. Результат отправки и длительность видны в `tripgo.position_requests` и `tripgo.position_requests.duration`, число сохранённых точек — в `tripgo.positions.saved`. Полученные команды можно проверить через `tripgoctl environment logs push-service`.

## Тесты

```sh
make test
make test-integration
```

`make test` запускает все тесты с детектором гонок и выводит покрытие Domain и Application. `make test-integration` запускает только интеграционные тесты. Обе команды требуют Docker: тесты создают отдельный PostgreSQL через Testcontainers, применяют миграции и удаляют контейнер после завершения.

Интеграционные тесты проверяют выборку устаревших поездок, HTTP-ручки маршрута и откат записи при ошибке. Тесты пула управляют тикером, проверяют дедупликацию, переполнение, отмену и остановку под нагрузкой. Утечки горутин проверяет goleak.

## Сборка

```sh
go build -o trips-service ./cmd/service
```

Для запуска бинарника переменные окружения должны быть заданы отдельно. OpenAPI-схема встроена в бинарник; наличие каталога `contracts/` в рабочей директории не требуется.
