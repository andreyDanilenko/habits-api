package logger

import (
	"backend/internal/model"
	"backend/internal/repository/logger"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Service struct {
	repo        *logger.Repository
	logDir      string
	currentFile *os.File
	currentDate string
	mu          sync.Mutex
}

func NewService(repo *logger.Repository, logDir string) *Service {
	// Создаем директорию для логов
	os.MkdirAll(logDir, 0755)

	return &Service{
		repo:   repo,
		logDir: logDir,
	}
}

// WriteLog записывает строку лога в файл
func (s *Service) WriteLog(logLine string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Проверяем, нужно ли создать новый файл для нового дня
	today := time.Now().Format("2006-01-02")
	if s.currentDate != today {
		if s.currentFile != nil {
			s.currentFile.Close()
		}
		filename := filepath.Join(s.logDir, "requests-"+today+".log")
		file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return // Игнорируем ошибки записи
		}
		s.currentFile = file
		s.currentDate = today
	}

	// Записываем в файл
	if s.currentFile != nil {
		s.currentFile.WriteString(logLine + "\n")
		s.currentFile.Sync() // Синхронизируем сразу
	}
}

// SyncToDB синхронизирует вчерашний лог-файл в БД
func (s *Service) SyncToDB() error {
	// Берем вчерашний день
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	filename := filepath.Join(s.logDir, "requests-"+yesterday+".log")

	fmt.Println("filename", filename)

	// Читаем файл
	data, err := os.ReadFile(filename)
	fmt.Println("SyncToDB", data, err)

	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	// Парсим и записываем в БД
	entries := parseLogFile(string(data))

	fmt.Println("SyncToDB", entries)

	if len(entries) > 0 {
		return s.repo.BatchInsert(entries)
	}

	return nil
}

// GetLogsByDate возвращает логи за определенную дату из БД
func (s *Service) GetLogsByDate(ctx context.Context, date time.Time) ([]*model.LogEntry, error) {
	return s.repo.GetLogsByDate(ctx, date)
}

// parseLogFile парсит файл логов
func parseLogFile(content string) []*model.LogEntry {
	entries := make([]*model.LogEntry, 0)

	// Разбиваем на строки
	lines := []string{}
	current := ""
	for _, char := range content {
		if char == '\n' {
			if current != "" {
				lines = append(lines, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}
	if current != "" {
		lines = append(lines, current)
	}

	// Парсим каждую строку
	for _, line := range lines {
		if line == "" {
			continue
		}
		entry := parseLogLine(line)
		if entry != nil {
			entries = append(entries, entry)
		}
	}

	return entries
}

// parseLogLine парсит одну строку: [GIN] 2006/01/02 - 15:04:05 | 200 | 4.83725ms | ::1 | GET      "/health"
func parseLogLine(line string) *model.LogEntry {
	// Разбиваем по "|"
	parts := []string{}
	current := ""
	for _, char := range line {
		if char == '|' {
			parts = append(parts, trimSpace(current))
			current = ""
		} else {
			current += string(char)
		}
	}
	if current != "" {
		parts = append(parts, trimSpace(current))
	}

	if len(parts) < 5 {
		return nil
	}

	// Парсим timestamp из первой части: [GIN] 2006/01/02 - 15:04:05
	timestampStr := parts[0]
	timestamp, err := time.Parse("[GIN] 2006/01/02 - 15:04:05", timestampStr)
	if err != nil {
		timestamp = time.Now() // Если не удалось распарсить, используем текущее время
	}

	// Парсим status code
	statusCode := parseInt(parts[1])

	// Парсим duration
	durationStr := parts[2]
	duration := parseDuration(durationStr)

	// IP
	clientIP := parts[3]

	// Method и path из последней части
	method, path := parseMethodPath(parts[4])

	return &model.LogEntry{
		Timestamp:  timestamp,
		StatusCode: statusCode,
		Duration:   duration,
		ClientIP:   clientIP,
		Method:     method,
		Path:       path,
		RawLog:     line,
	}
}

func trimSpace(s string) string {
	start := 0
	for start < len(s) && s[start] == ' ' {
		start++
	}
	end := len(s)
	for end > start && s[end-1] == ' ' {
		end--
	}
	return s[start:end]
}

func parseInt(s string) int {
	result := 0
	for _, char := range s {
		if char >= '0' && char <= '9' {
			result = result*10 + int(char-'0')
		}
	}
	return result
}

func parseDuration(s string) time.Duration {
	// Парсим "4.83725ms" или "12.875µs"
	var value float64
	divisor := 1.0
	decimal := false

	for _, char := range s {
		if char >= '0' && char <= '9' {
			if decimal {
				divisor *= 10
				value += float64(char-'0') / divisor
			} else {
				value = value*10 + float64(char-'0')
			}
		} else if char == '.' {
			decimal = true
		}
	}

	// Определяем единицу измерения
	if contains(s, "ms") {
		return time.Duration(value * 1000000)
	} else if contains(s, "µs") {
		return time.Duration(value * 1000)
	} else if contains(s, "ns") {
		return time.Duration(value)
	}
	return 0
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func parseMethodPath(s string) (string, string) {
	s = trimSpace(s)

	// Ищем первый пробел после метода
	methodEnd := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' {
			methodEnd = i
			break
		}
	}

	if methodEnd == 0 {
		return s, ""
	}

	return s[:methodEnd], trimSpace(s[methodEnd:])
}
