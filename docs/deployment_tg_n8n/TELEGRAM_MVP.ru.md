# Telegram MVP интеграция (готово к запуску)

Этот MVP делает простую цепочку:

`ERP (Go/Nest) -> n8n Webhook -> Telegram`

## 1) Что уже подготовлено

- Готовый workflow для импорта:
  - `deployment/n8n/workflows/telegram-mvp-webhook.json`
- Путь webhook в workflow:
  - `POST /webhook/erp-task-created`
- Проверка секрета:
  - `body.secret` должен совпасть с `N8N_TELEGRAM_WEBHOOK_SECRET`

## 2) Добавь переменные в `deployment/.env`

```env
N8N_TELEGRAM_WEBHOOK_SECRET=replace_with_strong_secret
TELEGRAM_BOT_TOKEN=your_bot_token
TELEGRAM_CHAT_ID=your_chat_id
```

## 3) Запусти/перезапусти n8n

```bash
cd deployment
docker compose --env-file .env up -d n8n
```

## 4) Импортируй workflow в n8n

1. Открой n8n UI.
2. `Workflows -> Import from File`.
3. Выбери `deployment/n8n/workflows/telegram-mvp-webhook.json`.
4. Активируй workflow.

## 5) Быстрый тест через curl

Локально:

```bash
curl -X POST "http://localhost:5678/webhook/erp-task-created" \
  -H "Content-Type: application/json" \
  -d '{
    "secret":"replace_with_strong_secret",
    "taskTitle":"Подготовить демо для CEO",
    "assignee":"Денис",
    "dueDate":"2026-03-25 18:00",
    "link":"https://habits.lifedream.tech/tasks/123"
  }'
```

Прод:

```bash
curl -X POST "https://n8n.lifedream.tech/webhook/erp-task-created" \
  -H "Content-Type: application/json" \
  -d '{
    "secret":"replace_with_strong_secret",
    "taskTitle":"Подготовить демо для CEO",
    "assignee":"Денис",
    "dueDate":"2026-03-25 18:00",
    "link":"https://habits.lifedream.tech/tasks/123"
  }'
```

Ожидаемый ответ:

```json
{"ok":true,"message":"Telegram notification sent"}
```

## 6) Пример вызова из Go (вставить в ERP service)

```go
package integrations

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type TelegramWebhookPayload struct {
	Secret    string `json:"secret"`
	TaskTitle string `json:"taskTitle"`
	Assignee  string `json:"assignee"`
	DueDate   string `json:"dueDate"`
	Link      string `json:"link"`
}

func SendTaskCreatedToN8N(webhookURL, secret, title, assignee, dueDate, link string) error {
	payload := TelegramWebhookPayload{
		Secret:    secret,
		TaskTitle: title,
		Assignee:  assignee,
		DueDate:   dueDate,
		Link:      link,
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest(http.MethodPost, webhookURL, bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("n8n webhook returned status %d", resp.StatusCode)
	}
	return nil
}
```

## 7) Что показывать CEO

- Создаешь/обновляешь задачу в ERP.
- ERP шлет webhook в n8n.
- В Telegram сотрудника приходит уведомление.

Это уже рабочая продажная ценность MVP.
