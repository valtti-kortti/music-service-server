# Music Room Service (Server)

Сервер для совместного прослушивания музыки с друзьями.  
Проект предоставляет HTTP API и WebSocket для управления «комнатами», очередями треков и синхронного воспроизведения треков YouTube на клиентских машинах.

Клиентская часть (интерфейс, который запускается у пользователя локально) находится здесь:  
https://github.com/valtti-kortti/music-service-client

---

## Возможности

- Создание комнат для совместного прослушивания
- Добавление треков в очередь по имени (поиск по YouTube)
- Удаление треков из очереди
- Управление воспроизведением в комнате:
    - Play / Pause
    - Next (следующий трек из очереди)
- Рассылка состояния комнаты всем подключённым клиентам через WebSocket:
    - текущий трек
    - очередь
    - флаг `playing`
    - позиция воспроизведения в секундах

Сервер **не стримит аудио сам** — он только координирует состояние.  
Каждый клиент сам запускает локальный плеер (`mpv`) и ходит по YouTube‑URL, которые отдаёт сервер.

---

## Архитектура (в общих чертах)

- `internal/audio` — сервис работы с YouTube API (поиск роликов, получение URL и длительности).
- `internal/room` — бизнес‑логика комнат:
    - очередь треков,
    - текущий трек,
    - состояние `playing` / `pause`,
    - расчёт текущей позиции,
    - подписчики (слушатели) и рассылка `State`.
- `internal/transport/http` — REST‑ручки (создание комнат, поиск видео, управление очередью и т.п.).
- `internal/transport/ws` — WebSocket‑подключение к комнате и стрим состояния.
- `cmd/server/main.go` — точка входа HTTP/WebSocket сервера.

---

## Требования

- Go 1.21+
- Аккаунт в Google Cloud и API ключ для **YouTube Data API v3**
- Docker (опционально, для контейнеризации)

---

## Переменные окружения

Сервер читает конфиг из переменных окружения (через `envconfig`) и при разработке может подхватывать `.env` через `godotenv`.

Пример `.env`:

```env
TOKEN=YOUR_YOUTUBE_API_KEY
LIMIT=10

READ_TIMEOUT=5
WRITE_TIMEOUT=10
READ_HEADER_TIMEOUT=2
IDLE_TIMEOUT=60

ADDRESS=:8080
```

Где:

- `TOKEN` — YouTube Data API ключ;
- `LIMIT` — максимальное количество результатов поиска видео;
- таймауты — в секундах;
- `ADDRESS` — адрес, на котором слушает HTTP сервер (например, `:8080`).

---

## Локальный запуск (без Docker)

```bash
# 1. Установить зависимости
go mod tidy

# 2. Создать .env рядом с main.go / Dockerfile
cp .env.example .env   # если есть пример, либо создать вручную

# 3. Запустить сервер
go run ./cmd/server
```

После старта сервер слушает по адресу `ADDRESS` из конфигурации, например:

```text
Listening on :8080
```

---

## Запуск в Docker

### Dockerfile (multi-stage)

В репозитории уже должен быть Dockerfile наподобие:

```dockerfile
FROM golang:1.21 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . ./

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server ./cmd/server

FROM alpine:3.19

WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY --from=builder /app/server ./server

ENV ADDRESS=:8080

EXPOSE 8080

CMD ["./server"]
```

### docker-compose.yml (пример)

```yaml
services:
  server:
    build: .
    container_name: music-room-server
    env_file:
      - .env
    ports:
      - "8080:8080"
```

Запуск:

```bash
docker compose build
docker compose up
```

Сервер будет доступен по адресу: `http://localhost:8080`.

---

## HTTP API (общая идея)

Ниже примерный набор эндпоинтов (конкретные пути могут отличаться от реализации):

### Поиск видео

```http
GET /api/v1/videos?name={query}
```

Ответ: список найденных видео (URL YouTube, название, длительность секундой).

### Работа с комнатами

```http
POST /api/v1/rooms
```

Создать комнату, вернуть `room_id`.

```http
POST 
```

Добавить трек в очередь.

Тело (JSON):

```json
{
  "url": "https://www.youtube.com/watch?v=...",
  "title": "Some title",
  "duration": 240
}
```

```http
DELETE /api/v1/rooms/delete?id={room_id}&idx={index}
```


Управление воспроизведением в комнате.


Ответ: текущий `State` комнаты (текущий трек, очередь, позиция, флаг `playing`).

---

## WebSocket

Подключение к комнате по WebSocket:

```text
ws://localhost:8080/ws/v1/rooms
```

После подключения сервер будет периодически отправлять JSON с состоянием комнаты, например:

```json
{
  "id": "f6f3b9ab-...",
  "current": {
    "url": "https://www.youtube.com/watch?v=...",
    "title": "Some track",
    "duration": 240
  },
  "queue": [
    { "url": "...", "title": "...", "duration": 200 }
  ],
  "playing": true,
  "position": 37.5,
  "updated_at": "2025-11-22T14:30:00Z"
}
```

Клиент (TUI/GUI) реагирует на это состояние и запускает/останавливает локальное воспроизведение через `mpv`.

---

## Клиентская часть

Репозиторий клиента:  
https://github.com/valtti-kortti/music-service-client

---

## Разработка

Рекомендуемый рабочий цикл:

1. Запустить сервер локально (`go run ./cmd/server`) или через Docker.
2. Поднять клиент (`music-service-client`) и прописать адрес сервера (`http://localhost:8080`).
3. Через клиент:
    - создать комнату,
    - добавить треки в очередь,
    - управлять воспроизведением (play/pause/next).
4. Допиливать бизнес‑логику (очередь, синхронизация, права) и расширять API.

PR‑ы и идеи по расширению API / протокола синхронизации приветствуются.