#!/bin/bash
# Генерирует файл targets.txt для vegeta
# Каждая строка - это HTTP запрос в формате vegeta

BASE_URL="${BASE_URL:-http://localhost:8080}"
NUM_REQUESTS="${NUM_REQUESTS:-300}"
TEAM_NAME="${TEAM_NAME:-load-team}"

TARGETS_FILE="load/targets.txt"
rm -f "$TARGETS_FILE"

echo "Генерируем $NUM_REQUESTS целей для vegeta..."

for i in $(seq 1 $NUM_REQUESTS); do
  pr_id="load-${i}-$(date +%s%N)"
  echo "POST ${BASE_URL}/pullRequest/create HTTP/1.1" >> "$TARGETS_FILE"
  echo "Host: localhost:8080" >> "$TARGETS_FILE"
  echo "Content-Type: application/json" >> "$TARGETS_FILE"
  echo "" >> "$TARGETS_FILE"
  echo "{\"pull_request_id\":\"${pr_id}\",\"pull_request_name\":\"Load test PR ${i}\",\"author_id\":\"lu1\"}" >> "$TARGETS_FILE"
  echo "" >> "$TARGETS_FILE"
done

echo "Файл $TARGETS_FILE создан с $NUM_REQUESTS целями."







