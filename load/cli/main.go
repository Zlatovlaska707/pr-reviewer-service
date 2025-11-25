package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	vegeta "github.com/tsenart/vegeta/v12/lib"
)

const (
	defaultBaseURL     = "http://localhost:8080"
	defaultRate        = 5
	defaultDuration    = 60 * time.Second
	defaultTeamName    = "load-team"
	defaultResultsFile = "load/artifacts/results.bin"
)

var resultsFile = defaultResultsFile

func main() {
	var (
		baseURL   = flag.String("url", defaultBaseURL, "Base URL сервиса")
		rate      = flag.Int("rate", defaultRate, "Запросов в секунду")
		duration  = flag.Duration("duration", defaultDuration, "Длительность теста (например, 60s)")
		teamName  = flag.String("team", defaultTeamName, "Имя тестовой команды")
		setupOnly = flag.Bool("setup-only", false, "Только подготовка окружения (создание команды)")
		report    = flag.Bool("report", false, "Показать отчёт из сохранённых результатов")
		plot      = flag.Bool("plot", false, "Сгенерировать HTML график из сохранённых результатов")
	)
	flag.Parse()

	if *report {
		showReport()
		return
	}

	if *plot {
		generatePlot()
		return
	}

	if *setupOnly {
		if err := setupTeam(*baseURL, *teamName); err != nil {
			log.Fatalf("Ошибка при подготовке окружения: %v", err)
		}
		return
	}

	// Полный цикл: setup + нагрузочное тестирование
	fmt.Println("=== Нагрузочное тестирование с Vegeta ===")
	fmt.Printf("URL: %s\n", *baseURL)
	fmt.Printf("Rate: %d req/s\n", *rate)
	fmt.Printf("Duration: %s\n", *duration)
	fmt.Println()

	fmt.Println("1. Подготовка тестового окружения...")
	if err := setupTeam(*baseURL, *teamName); err != nil {
		log.Fatalf("Ошибка при подготовке окружения: %v", err)
	}

	fmt.Println()
	fmt.Println("2. Запуск нагрузочного тестирования...")
	if err := runLoadTest(*baseURL, *rate, *duration, *teamName); err != nil {
		log.Fatalf("Ошибка при нагрузочном тестировании: %v", err)
	}

	fmt.Println()
	fmt.Println("=== Тестирование завершено ===")
	fmt.Println("Для детального анализа выполните:")
	fmt.Printf("  go run ./load/cli -report\n")
	fmt.Printf("  go run ./load/cli -plot\n")
}

// setupTeam создаёт тестовую команду с пользователями
func setupTeam(baseURL, teamName string) error {
	payload := map[string]interface{}{
		"team_name": teamName,
		"members": []map[string]interface{}{
			{"user_id": "lu1", "username": "Load Alice", "is_active": true},
			{"user_id": "lu2", "username": "Load Bob", "is_active": true},
			{"user_id": "lu3", "username": "Load Carol", "is_active": true},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	// Используем vegeta для отправки одного запроса setup
	targeter := vegeta.NewStaticTargeter(vegeta.Target{
		Method: "POST",
		URL:    fmt.Sprintf("%s/team/add", baseURL),
		Header: http.Header{"Content-Type": []string{"application/json"}},
		Body:   body,
	})

	attacker := vegeta.NewAttacker()
	var metrics vegeta.Metrics

	for res := range attacker.Attack(targeter, vegeta.Rate{Freq: 1, Per: time.Second}, time.Second, "setup") {
		metrics.Add(res)
	}
	metrics.Close()

	if metrics.StatusCodes["201"] == 0 && metrics.StatusCodes["400"] == 0 {
		return fmt.Errorf("не удалось создать команду: статус %v", metrics.StatusCodes)
	}

	fmt.Printf("Команда '%s' создана (или уже существовала)\n", teamName)
	return nil
}

// runLoadTest запускает нагрузочное тестирование
func runLoadTest(baseURL string, rate int, duration time.Duration, teamName string) error {
	if rate <= 0 {
		return fmt.Errorf("rate must be positive, got %d", rate)
	}
	targeter := newPullRequestTargeter(baseURL)

	// Настраиваем атакующего
	workers := uint64(rate)
	attacker := vegeta.NewAttacker(
		vegeta.Timeout(30*time.Second),
		vegeta.Workers(workers),
	)

	// Запускаем атаку
	var metrics vegeta.Metrics
	ctx, cancel := context.WithTimeout(context.Background(), duration+5*time.Second)
	defer cancel()

	rateLimit := vegeta.Rate{Freq: rate, Per: time.Second}
	results := attacker.Attack(targeter, rateLimit, duration, "load-test")

	// Собираем результаты
	var allResults []vegeta.Result
	for res := range results {
		select {
		case <-ctx.Done():
			break
		default:
			metrics.Add(res)
			allResults = append(allResults, *res)
		}
	}
	metrics.Close()

	// Сохраняем результаты в файл
	if err := saveResults(allResults); err != nil {
		return fmt.Errorf("сохранить результаты: %w", err)
	}

	// Выводим отчёт
	reporter := vegeta.NewTextReporter(&metrics)
	if err := reporter(os.Stdout); err != nil {
		return fmt.Errorf("сгенерировать отчёт: %w", err)
	}

	return nil
}

func newPullRequestTargeter(baseURL string) vegeta.Targeter {
	return func(t *vegeta.Target) error {
		payload := map[string]any{
			"pull_request_id": fmt.Sprintf("load-%d", time.Now().UnixNano()),
			"pull_request_name": fmt.Sprintf(
				"Load test PR %s",
				time.Now().Format(time.RFC3339Nano),
			),
			"author_id": "lu1",
		}

		body, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal payload: %w", err)
		}

		*t = vegeta.Target{
			Method: "POST",
			URL:    fmt.Sprintf("%s/pullRequest/create", baseURL),
			Header: http.Header{"Content-Type": []string{"application/json"}},
			Body:   body,
		}

		return nil
	}
}

// saveResults сохраняет результаты в бинарный файл
func saveResults(results []vegeta.Result) error {
	if err := os.MkdirAll(filepath.Dir(resultsFile), 0o755); err != nil {
		return fmt.Errorf("создать директорию: %w", err)
	}

	file, err := os.Create(resultsFile)
	if err != nil {
		return fmt.Errorf("создать файл: %w", err)
	}
	defer file.Close()

	encoder := vegeta.NewEncoder(file)
	for i := range results {
		if err := encoder.Encode(&results[i]); err != nil {
			return fmt.Errorf("записать результат: %w", err)
		}
	}

	fmt.Printf("Результаты сохранены в %s\n", resultsFile)
	return nil
}

// showReport показывает отчёт из сохранённых результатов
func showReport() {
	if err := renderReport(os.Stdout, resultsFile); err != nil {
		log.Fatalf("Не удалось построить отчёт: %v", err)
	}
}

func renderReport(out io.Writer, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open results: %w", err)
	}
	defer file.Close()

	decoder := vegeta.NewDecoder(file)
	var metrics vegeta.Metrics

	for {
		var res vegeta.Result
		if err := decoder.Decode(&res); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("decode result: %w", err)
		}
		metrics.Add(&res)
	}
	metrics.Close()

	reporter := vegeta.NewTextReporter(&metrics)
	if err := reporter(out); err != nil {
		return fmt.Errorf("render report: %w", err)
	}
	return nil
}

// generatePlot генерирует HTML график из сохранённых результатов
// Использует CLI утилиту vegeta для генерации графика
func generatePlot() {
	writePlotInstructions(os.Stdout)
}

func writePlotInstructions(out io.Writer) {
	fmt.Fprintln(out, "Для генерации HTML графика используйте CLI утилиту vegeta:")
	fmt.Fprintf(out, "  vegeta plot %s > load/artifacts/plot.html\n", resultsFile)
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Установка CLI утилиты:")
	fmt.Fprintln(out, "  go install github.com/tsenart/vegeta/v12@latest")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Или используйте сохранённые результаты для анализа через другие инструменты.")
}
