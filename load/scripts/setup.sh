#!/bin/bash
# Скрипт для подготовки тестового окружения перед нагрузочным тестированием

BASE_URL="${BASE_URL:-http://localhost:8080}"
TEAM_NAME="${TEAM_NAME:-load-team}"

echo "Создаём тестовую команду: $TEAM_NAME"

curl -X POST "${BASE_URL}/team/add" \
  -H "Content-Type: application/json" \
  -d "{
    \"team_name\": \"${TEAM_NAME}\",
    \"members\": [
      {\"user_id\": \"lu1\", \"username\": \"Load Alice\", \"is_active\": true},
      {\"user_id\": \"lu2\", \"username\": \"Load Bob\", \"is_active\": true},
      {\"user_id\": \"lu3\", \"username\": \"Load Carol\", \"is_active\": true}
    ]
  }"

echo ""
echo "Команда создана. Можно запускать нагрузочное тестирование."







