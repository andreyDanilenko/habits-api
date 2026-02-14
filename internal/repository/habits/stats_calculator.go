package habits

import (
	"time"
)

// StatsCalculator вычисляет статистику привычек
type StatsCalculator struct{}

// HabitScheduleInfo данные расписания привычки для расчёта streaks
type HabitScheduleInfo struct {
	ScheduleType  string
	RecurringDays []int
	OneTimeDate   *time.Time
	CreatedAtUTC  time.Time
}

// CalculateStreaks вычисляет currentStreak и longestStreak
func (c *StatsCalculator) CalculateStreaks(completionDates []time.Time, info HabitScheduleInfo) (currentStreak, longestStreak int) {
	if len(completionDates) == 0 {
		return 0, 0
	}

	todayNormalized := NormalizeDate(time.Now().UTC())
	completionMap := make(map[string]bool)
	for _, d := range completionDates {
		completionMap[NormalizeDate(d).Format("2006-01-02")] = true
	}

	isHabitActiveOnDate := func(date time.Time) bool {
		nd := NormalizeDate(date)
		if nd.Before(info.CreatedAtUTC) {
			return false
		}
		if info.ScheduleType == "recurring" {
			dayOfWeek := int(nd.Weekday())
			for _, day := range info.RecurringDays {
				if day == dayOfWeek {
					return true
				}
			}
			return false
		}
		if info.ScheduleType == "one_time" && info.OneTimeDate != nil {
			return nd.Equal(NormalizeDate(*info.OneTimeDate))
		}
		return false
	}

	// currentStreak
	checkDate := todayNormalized
	for {
		if !isHabitActiveOnDate(checkDate) {
			break
		}
		if completionMap[checkDate.Format("2006-01-02")] {
			currentStreak++
			checkDate = checkDate.AddDate(0, 0, -1)
		} else {
			break
		}
	}

	// longestStreak
	var activeDates []time.Time
	for _, d := range completionDates {
		if isHabitActiveOnDate(d) {
			activeDates = append(activeDates, d)
		}
	}
	if len(activeDates) == 0 {
		return currentStreak, 0
	}

	sorted := make([]time.Time, len(activeDates))
	copy(sorted, activeDates)
	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if NormalizeDate(sorted[j]).After(NormalizeDate(sorted[j+1])) {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	longestStreak = 0
	currentSeq := 0
	var prevDate time.Time
	for _, date := range sorted {
		nd := NormalizeDate(date)
		if prevDate.IsZero() {
			currentSeq = 1
			prevDate = nd
		} else {
			if nd.Equal(prevDate.AddDate(0, 0, 1)) {
				currentSeq++
			} else {
				if currentSeq > longestStreak {
					longestStreak = currentSeq
				}
				currentSeq = 1
			}
			prevDate = nd
		}
	}
	if currentSeq > longestStreak {
		longestStreak = currentSeq
	}

	return currentStreak, longestStreak
}
