#!/bin/bash
# Основной скрипт для запуска нагрузочного тестирования с vegeta

BASE_URL="${BASE_URL:-http://localhost:8080}"
RATE="${RATE:-5}"           # запросов в секунду (соответствует требованию ТЗ)
DURATION="${DURATION:-60s}"  # длительность теста
TEAM_NAME="${TEAM_NAME:-load-team}"

echo "=== Нагрузочное тестирование с Vegeta ==="
echo "URL: $BASE_URL"
echo "Rate: $RATE req/s"
echo "Duration: $DURATION"
echo ""

# Проверяем наличие vegeta
if ! command -v vegeta &> /dev/null; then
    echo "Ошибка: vegeta не установлен."
    echo "Установите: go install github.com/tsenart/vegeta/v12@latest"
    exit 1
fi

# Подготавливаем окружение
echo "1. Подготовка тестового окружения..."
bash load/setup.sh

# Генерируем targets
echo ""
echo "2. Генерация целей для тестирования..."
bash load/generate_targets.sh

# Запускаем vegeta и сохраняем результаты
RESULTS_FILE="load/results.bin"
echo ""
echo "3. Запуск нагрузочного тестирования..."
echo "POST ${BASE_URL}/pullRequest/create" | \
  vegeta attack \
    -rate="$RATE" \
    -duration="$DURATION" \
    -header="Content-Type: application/json" \
    -body='{"pull_request_id":"load-{{.TimestampNano}}","pull_request_name":"Load test PR","author_id":"lu1"}' \
    > "$RESULTS_FILE"

# Показываем отчёт
echo ""
echo "4. Результаты тестирования:"
vegeta report "$RESULTS_FILE"

echo ""
echo "=== Тестирование завершено ==="
echo "Для детального анализа выполните:"
echo "  vegeta report load/results.bin"
echo "  vegeta plot load/results.bin > load/plot.html"

