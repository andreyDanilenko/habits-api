package worker

import (
	"backend/internal/service/logger"
	"context"
	"log"
	"time"
)

// LogProcessor синхронизирует логи в БД раз в сутки
type LogProcessor struct {
	logService *logger.Service
	stopChan   chan struct{}
}

func NewLogProcessor(logService *logger.Service) *LogProcessor {
	return &LogProcessor{
		logService: logService,
		stopChan:   make(chan struct{}),
	}
}

// Start запускает синхронизацию раз в сутки в полночь
func (w *LogProcessor) Start(ctx context.Context) {
	go func() {
		// Вычисляем время до следующей полночи
		now := time.Now()
		nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
		initialDelay := nextMidnight.Sub(now)

		log.Printf("LogProcessor: синхронизация начнется через %v", initialDelay)

		// Первый запуск в полночь
		timer := time.NewTimer(initialDelay)
		select {
		case <-timer.C:
			w.sync()
			// Затем каждые 24 часа
			ticker := time.NewTicker(24 * time.Hour)
			for {
				select {
				case <-ticker.C:
					w.sync()
				case <-w.stopChan:
					ticker.Stop()
					return
				case <-ctx.Done():
					ticker.Stop()
					return
				}
			}
		case <-w.stopChan:
			timer.Stop()
			return
		case <-ctx.Done():
			timer.Stop()
			return
		}
	}()
}

// Stop останавливает воркер
func (w *LogProcessor) Stop() {
	close(w.stopChan)
}

// sync синхронизирует вчерашние логи в БД
func (w *LogProcessor) sync() {
	log.Println("LogProcessor: синхронизация логов в БД...")
	if err := w.logService.SyncToDB(); err != nil {
		log.Printf("LogProcessor: ошибка синхронизации: %v", err)
	} else {
		log.Println("LogProcessor: синхронизация завершена")
	}
}
