package habits

import (
	"regexp"
	"time"

	"github.com/lib/pq"
)

// NormalizeDate приводит дату к UTC и началу дня
func NormalizeDate(t time.Time) time.Time {
	utc := t.UTC()
	year, month, day := utc.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

// ConvertPreferredTimeToTime конвертирует строковое значение preferredTime в формат времени для PostgreSQL
// "morning" -> "08:00:00", "afternoon" -> "14:00:00", "evening" -> "20:00:00"
// Если значение уже в формате времени (HH:MM:SS), возвращает его как есть
func ConvertPreferredTimeToTime(preferredTime string) string {
	switch preferredTime {
	case "morning":
		return "08:00:00"
	case "afternoon":
		return "14:00:00"
	case "evening":
		return "20:00:00"
	default:
		if matched, _ := regexp.MatchString(`^\d{1,2}:\d{2}(:\d{2})?$`, preferredTime); matched {
			return preferredTime
		}
		return "08:00:00"
	}
}

// ConvertTimeToPreferredTime конвертирует время из БД обратно в строковое значение для фронтенда
// "08:00:00" -> "morning", "14:00:00" -> "afternoon", "20:00:00" -> "evening"
func ConvertTimeToPreferredTime(timeStr string) string {
	switch timeStr {
	case "08:00:00", "08:00":
		return "morning"
	case "14:00:00", "14:00":
		return "afternoon"
	case "20:00:00", "20:00":
		return "evening"
	default:
		return timeStr
	}
}

// ConvertRecurringDays конвертирует pq.Int32Array в []int
func ConvertRecurringDays(daysArray pq.Int32Array) []int {
	if daysArray == nil {
		return nil
	}
	result := make([]int, len(daysArray))
	for i, v := range daysArray {
		result[i] = int(v)
	}
	return result
}
