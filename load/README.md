# Нагрузочное тестирование с Vegeta

Этот проект использует библиотеку [Vegeta](https://github.com/tsenart/vegeta) для нагрузочного тестирования API сервиса назначения ревьюверов. Тестирование реализовано как Go-программа в `load/cli/`.

## Быстрый старт

1. Запустите сервис:
   ```bash
   docker compose up --build
   ```

2. Запустите нагрузочное тестирование:
   ```bash
   make load-test
   ```

   Или напрямую:
   ```bash
   go run ./load/cli
   ```

## Структура

- `load/cli/` — Go-приложение для нагрузочного тестирования
  - `main.go` — основной код (setup, генерация запросов, запуск Vegeta)
  - `main_test.go` — unit тесты
- `load/scripts/` — вспомогательные shell-скрипты (устаревшие, используются для справки)
  - `setup.sh` — подготовка тестового окружения
  - `load_test.sh` — запуск нагрузочного теста
  - `generate_targets.sh` — генерация targets для Vegeta
- `load/artifacts/` — результаты и артефакты тестов
  - `results.bin` — бинарный файл с результатами теста
  - `plot.html` — HTML график результатов (генерируется отдельно)
  - `vegeta-plot.png` — визуализация результатов

## Параметры

Можно настроить через флаги командной строки:

```bash
go run ./load/cli -url=http://localhost:8080 -rate=5 -duration=60s -team=load-team
```

### Доступные флаги

| Флаг | Описание | Значение по умолчанию |
|------|----------|----------------------|
| `-url` | Base URL сервиса | `http://localhost:8080` |
| `-rate` | Запросов в секунду | `5` |
| `-duration` | Длительность теста | `60s` |
| `-team` | Имя тестовой команды | `load-team` |
| `-setup-only` | Только подготовка окружения (создание команды) | `false` |
| `-report` | Показать отчёт из сохранённых результатов | `false` |
| `-plot` | Показать инструкцию по генерации графика | `false` |

### Примеры использования

```bash
# Кастомные параметры
go run ./load/cli -rate=10 -duration=30s

# Только подготовка окружения
go run ./load/cli -setup-only

# Показать отчёт из сохранённых результатов
go run ./load/cli -report

# Показать инструкцию по генерации графика
go run ./load/cli -plot
```

## Команды Makefile

| Команда | Описание |
|---------|----------|
| `make load-test` | Запустить полный цикл тестирования (setup + нагрузка) |
| `make load-test-setup` | Только подготовка окружения (создание команды) |
| `make load-test-report` | Показать отчёт из сохранённых результатов |
| `make load-test-plot` | Показать инструкцию по генерации графика |

## Сценарий тестирования

1. **Подготовка**: автоматически создаётся команда `load-team` с тремя активными пользователями:
   - `lu1` — Load Alice
   - `lu2` — Load Bob
   - `lu3` — Load Carol

2. **Нагрузка**: Vegeta отправляет POST запросы на `/pullRequest/create` с уникальными ID (генерируются на основе `time.Now().UnixNano()` для избежания конфликтов)

3. **Проверка**: автоматически назначаются ревьюверы из команды автора (до 2 штук)

## Анализ результатов

После теста результаты сохраняются в `load/artifacts/results.bin`. Для анализа:

### Текстовый отчёт

```bash
# Через Go-программу
go run ./load/cli -report

# Или через CLI утилиту vegeta (если установлена)
vegeta report load/artifacts/results.bin
```

### HTML график

```bash
# Установить CLI утилиту vegeta
go install github.com/tsenart/vegeta/v12@latest

# Сгенерировать график
vegeta plot load/artifacts/results.bin > load/artifacts/plot.html

# Открыть в браузере
open load/artifacts/plot.html  # macOS
xdg-open load/artifacts/plot.html  # Linux
```

### Детальная статистика в JSON

```bash
vegeta report -type=json load/artifacts/results.bin | jq
```

## Требования SLA

Согласно ТЗ:
- **RPS**: 5 запросов в секунду
- **SLI времени ответа**: 300 мс (95-й перцентиль)
- **SLI успешности**: 99.9%

Vegeta автоматически проверяет эти метрики и выводит отчёт. Подробные результаты см. в `load/load-test-report.md`.

