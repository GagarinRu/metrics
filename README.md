# Metrics Collection Service

![Go](https://img.shields.io/badge/Go-1.25-blue?logo=go)
![Chi](https://img.shields.io/badge/Chi-5.2.5-blue?logo=go)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-17-orange?logo=postgresql)
![Docker](https://img.shields.io/badge/Docker-24.0-blue?logo=docker)
![Zap](https://img.shields.io/badge/Zap-1.27-green?logo=go)


## Описание

Сервис сбора метрик предназначен для сбора и хранения метрик операционной системы (CPU, memory, disk, network). Поддерживает как сбор метрик в файл, так и хранение в базе данных PostgreSQL.

## Основные функции

- **server**: HTTP-сервер для приёма метрик от агентов.
- **agent**: Агент для сбора метрик операционной системы и отправки на сервер.
- Поддержка сжатия данных (gzip).
- Хранение метрик в файле (JSON) или PostgreSQL.
- Вычисление хеша для проверки целостности данных.
- Пакетная отправка метрик.


## Технологический стек
- **Go** — основной язык программирования.
- **Chi** — роутер для HTTP-сервера.
- **pgx** — драйвер для PostgreSQL.
- **zap** — логирование.
- **gopsutil** — сбор метрик операционной системы.


## Требования
- Docker и Docker Compose
- Go 1.25+
- PostgreSQL 13+


## Инструкция по запуску

### 1. Подготовка

1. Склонируйте репозиторий:
```bash
git clone https://github.com/GagarinRu/metrics.git
cd metrics
```

2. Настройте параметры подключения к БД в файле `.env`:
```bash
cp .env.example .env
```

### 2. Запуск контейнеров

1. Убедитесь, что Docker и Docker Compose установлены.
2. Запустите сервисы с помощью Docker Compose:
```bash
docker-compose -f docker-compose.yml up --build
```

### 3. Запуск сервера

```bash
go run ./cmd/server -a :8080 -l info
```

Параметры сервера:
- `-a` — адрес сервера (по умолчанию `:8080`)
- `-l` — уровень логирования (по умолчанию `info`)
- `-i` — интервал сохранения метрик в файл в секундах (по умолчанию 300)
- `-f` — путь к файлу для хранения метрик (по умолчанию `metrics.json`)
- `-r` — восстановить метрики из файла при запуске
- `-d` — DSN для подключения к PostgreSQL
- `-k` — ключ для вычисления хеша

### 4. Запуск агента

```bash
go run ./cmd/agent -a http://localhost:8080
```

Параметры агента:
- `-a` — адрес сервера (по умолчанию `http://localhost:8080`)
- `-p` — интервал опроса метрик в секундах (по умолчанию 2)
- `-r` — интервал отправки метрик в секундах (по умолчанию 10)
- `-l` — уровень логирования (по умолчанию `info`)
- `-k` — ключ для вычисления хеша
- `-rate-limit` — ограничение параллельных запросов (по умолчанию 1)


## API эндпоинты

### Получить все метрики

```bash
GET /
```

### Получить конкретную метрику

```bash
GET /value/{metricType}/{metricName}
```

Где `metricType` — `gauge` или `counter`, `metricName` — имя метрики.

### Обновить метрику

```bash
POST /update/{metricType}/{metricName}/{metricValue}
```

### Обновить метрику (JSON)

```bash
POST /update
```

Пример запроса:
```json
{
  "id": "Alloc",
  "type": "gauge",
  "value": 123.45
}
```

### Пакетное обновление метрик

```bash
POST /updates
```

Пример запроса:
```json
[
  {"id": "Alloc", "type": "gauge", "value": 123.45},
  {"id": "BucketchSys", "type": "gauge", "value": 678.90}
]
```

### Получить метрику (JSON)

```bash
POST /value
```

Пример запроса:
```json
{
  "id": "Alloc",
  "type": "gauge"
}
```

### Проверить соединение с БД

```bash
GET /ping
```


## Пример использования

### Запуск с файловым хранилищем

```bash
go run ./cmd/server -a :8080 -f metrics.json -r
```

В другом терминале запустите агент:
```bash
go run ./cmd/agent -a http://localhost:8080 -p 2 -r 10
```

### Запуск с PostgreSQL

```bash
go run ./cmd/server -a :8080 -d "postgres://postgres:postgres@localhost:5432/metric?sslmode=disable"
```

### Проверка работы

Получите все метрики:
```bash
curl http://localhost:8080/
```

Получите конкретную метрику:
```bash
curl http://localhost:8080/value/gauge/Alloc
```


## Тестирование

Для запуска тестов:
```bash
go test -v ./...
```

Для запуска интеграционных тестов с БД:
```bash
docker-compose -f docker-compose.yml up --build
go test -v -tags=integration ./...
```


## Автор

[Evgeny Kudryashov](https://github.com/GagarinRu)