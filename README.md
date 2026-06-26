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

Покрытие кода тестами:
```bash
go test -coverprofile coverage.out ./...
go tool cover -func coverage.out
```

Для запуска интеграционных тестов с БД:
```bash
docker-compose -f docker-compose.yml up --build
go test -v -tags=integration ./...
```

## Бенчмарки

Бенчмарки измеряют скорость ключевых компонентов: хранилище, HTTP-обработчики, агент и сбор метрик.

```bash
go test -bench=. -benchmem ./internal/storage/ ./internal/handler/ ./internal/agent/ ./internal/metrics/ ./internal/profile/
```

| Компонент | Бенчмарк | Что измеряет |
|-----------|----------|--------------|
| storage | `BenchmarkUpdateGauge`, `BenchmarkUpdateBatch`, `BenchmarkSave` | Запись и сохранение метрик |
| handler | `BenchmarkUpdateMetricsBatch`, `BenchmarkGetAllMetrics`, `BenchmarkCalculateHash` | HTTP-обработка запросов |
| agent | `BenchmarkCollectAllMetrics`, `BenchmarkCalculateHash` | Сбор и подготовка метрик к отправке |
| metrics | `BenchmarkUpdateRuntimeMetrics`, `BenchmarkUpdateSystemMetrics` | Сбор runtime- и OS-метрик |
| profile | `BenchmarkSystemLoad` | Комплексная нагрузка (batch + чтение + HTML) |

## Профилирование памяти (pprof)

Сервер подключает `net/http/pprof` — профили доступны по адресу `/debug/pprof/` при запущенном сервере.

### Снятие профиля под нагрузкой

1. Запустите сервер и сгенерируйте нагрузку (например, `hey`):
```bash
go run ./cmd/server -a :8080 &
hey -z 30s -c 10 http://localhost:8080/
curl -o profiles/base.pprof http://localhost:8080/debug/pprof/heap
```

2. Для воспроизводимого профиля в тестах:
```bash
go test -bench=BenchmarkSystemLoad -benchmem -memprofile=profiles/base.pprof ./internal/profile/
# ... после оптимизаций ...
go test -bench=BenchmarkSystemLoad -benchmem -memprofile=profiles/result.pprof ./internal/profile/
```

### Анализ профиля

```bash
go tool pprof -top profiles/base.pprof
go tool pprof -list=GetAllMetrics profiles/base.pprof
go tool pprof -http=:9090 profiles/base.pprof
```

### Оптимизации

По результатам анализа `profiles/base.pprof` были устранены избыточные аллокации:

- `GetAllMetrics` — прямая запись HTML через `WriteMetricsHTML` без промежуточных map
- `calculateHash` — хеширование через `hash.Hash` вместо `append(data, key...)`
- `CollectAll` — сбор метрик агентом за одну блокировку без копирования map
- `UpdateSystemMetrics` — запись в map без промежуточной структуры
- `Save` — `json.Marshal` вместо `json.MarshalIndent`
- `GetAllGauges`/`GetAllCounters` — предварительное выделение ёмкости map

### Сравнение профилей

```bash
go tool pprof -top -diff_base=profiles/base.pprof profiles/result.pprof
```

Результат сравнения (отрицательные значения — снижение потребления памяти):

```
      flat  flat%   sum%        cum   cum%
   -7.50MB  6.61% 12.84%    -7.50MB  6.61%  github.com/GagarinRu/metrics/internal/storage.(*MemStorage).GetAllCounters
      -3MB  2.64% 16.39%   -12.01MB 10.58%  github.com/GagarinRu/metrics/internal/handler.(*Handler).GetAllMetrics
   -0.99MB  0.87% 31.88%    -0.99MB  0.87%  github.com/GagarinRu/metrics/internal/storage.(*MemStorage).GetAllGauges
```

Бенчмарк `BenchmarkSystemLoad` после оптимизации: **76590 → 65678 B/op**, **507 → 374 allocs/op**.


## Автор

[Evgeny Kudryashov](https://github.com/GagarinRu)