// seed_crm создаёт тестовые данные CRM для указанного workspace и пользователя (owner).
//
// Использование:
//
//	WORKSPACE_ID=<uuid> USER_ID=<uuid> go run ./cmd/seed_crm
//
// Пример:
//
//	 WORKSPACE_ID=4cb4b10e-f90e-4dbe-8a8a-f224efdebb96 USER_ID=e8694a42-d2d8-448b-8d22-183bd51823ce go run ./cmd/seed_crm
//		WORKSPACE_ID=59ebb151-036a-4d54-a5fb-a6d32466e8bd USER_ID=b37735c9-8934-4f3d-877c-bb5015e83b7e go run ./cmd/seed_crm
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"backend/internal/config"
	"backend/internal/database"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

func main() {
	workspaceID := os.Getenv("WORKSPACE_ID")
	userID := os.Getenv("USER_ID")
	if workspaceID == "" || userID == "" {
		log.Fatal("WORKSPACE_ID and USER_ID environment variables are required. Example: WORKSPACE_ID=550e8400-e29b-41d4-a716-446655440000 USER_ID=6ba7b810-9dad-11d1-80b4-00c04fd430c8 go run ./cmd/seed_crm")
	}
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		log.Fatalf("Invalid WORKSPACE_ID: %v", err)
	}
	uID, err := uuid.Parse(userID)
	if err != nil {
		log.Fatalf("Invalid USER_ID: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	db, err := database.InitDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// 1. Pipelines + stages (несколько воронок с разными этапами и цветами)
	type stageRow struct {
		name        string
		order       int
		probability int
		isFinal     bool
		isLost      bool
		color       string
	}
	pipelinesData := []struct {
		name      string
		isDefault bool
		stages    []stageRow
	}{
		{
			"Продажи",
			true,
			[]stageRow{
				{"Лиды", 1, 10, false, false, "#94A3B8"},
				{"Квалификация", 2, 30, false, false, "#64748B"},
				{"Предложение", 3, 60, false, false, "#3B82F6"},
				{"Переговоры", 4, 80, false, false, "#8B5CF6"},
				{"Сделка", 5, 100, true, false, "#22C55E"},
				{"Отказ", 6, 0, true, true, "#EF4444"},
			},
		},
		{
			"Подписки",
			false,
			[]stageRow{
				{"Триал", 1, 5, false, false, "#F59E0B"},
				{"Активный", 2, 50, false, false, "#06B6D4"},
				{"Продление", 3, 90, false, false, "#10B981"},
				{"Продлён", 4, 100, true, false, "#22C55E"},
				{"Отмена", 5, 0, true, true, "#EF4444"},
			},
		},
		{
			"Партнёрские",
			false,
			[]stageRow{
				{"Заявка", 1, 10, false, false, "#A78BFA"},
				{"Проверка", 2, 40, false, false, "#F472B6"},
				{"Договор", 3, 70, false, false, "#38BDF8"},
				{"Выплата", 4, 100, true, false, "#22C55E"},
				{"Отказ", 5, 0, true, true, "#EF4444"},
			},
		},
		{
			"B2B",
			false,
			[]stageRow{
				{"Квалификация", 1, 5, false, false, "#F59E0B"},
				{"Демо", 2, 30, false, false, "#06B6D4"},
				{"Оффер", 3, 60, false, false, "#8B5CF6"},
				{"Согласование", 4, 85, false, false, "#3B82F6"},
				{"Закрыто", 5, 100, true, false, "#22C55E"},
				{"Проигрыш", 6, 0, true, true, "#EF4444"},
			},
		},
	}
	pipeIDs := make([]uuid.UUID, len(pipelinesData))
	allStageIDs := make([][]uuid.UUID, len(pipelinesData)) // [pipelineIdx][stageIdx]
	for pIdx, pipe := range pipelinesData {
		pipeIDs[pIdx] = uuid.New()
		_, err = db.ExecContext(ctx, `INSERT INTO crm_pipelines (id, workspace_id, name, is_default, created_by, created_at) VALUES ($1,$2,$3,$4,$5,NOW())`,
			pipeIDs[pIdx], wsID, pipe.name, pipe.isDefault, uID)
		if err != nil {
			log.Fatalf("Insert pipeline %s: %v", pipe.name, err)
		}
		allStageIDs[pIdx] = make([]uuid.UUID, len(pipe.stages))
		for sIdx, s := range pipe.stages {
			allStageIDs[pIdx][sIdx] = uuid.New()
			_, err = db.ExecContext(ctx, `INSERT INTO crm_stages (id, pipeline_id, name, order_index, color, probability, is_final, is_lost, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW())`,
				allStageIDs[pIdx][sIdx], pipeIDs[pIdx], s.name, s.order, s.color, s.probability, s.isFinal, s.isLost)
			if err != nil {
				log.Fatalf("Insert stage %s: %v", s.name, err)
			}
		}
	}

	// 2. Companies (30) — все поля: inn, kpp, ogrn, phone, email, website, legal_address, actual_address, tags
	companyNames := []string{
		"ООО Ромашка", "ИП Иванов", "АО Северсталь", "ООО Технопарк", "ЗАО Мегафон",
		"ООО Газпром нефть", "ИП Петрова", "ООО Роснефть", "АО Сбербанк", "ООО Яндекс",
		"ИП Козлов", "ООО Лукойл", "ЗАО Магнит", "ООО Х5 Ритейл", "АО Норникель",
		"ООО Транснефть", "ИП Сидорова", "ООО Новатэк", "ООО Татнефть", "АО Газпром",
		"ООО Башнефть", "ИП Федоров", "ООО Русал", "ООО НЛМК", "АО Полюс",
		"ООО Сургутнефтегаз", "ИП Морозов", "ООО Северсталь", "ООО Металлоинвест", "АО НЛМК",
	}
	companyTags := [][]string{
		{"ключевой", "розница"}, {"малый бизнес"}, {"крупный", "металл"}, {"ит", "стартап"}, {"телеком"},
		{"нефть"}, {"малый бизнес"}, {"нефть", "крупный"}, {"финансы"}, {"ит", "крупный"},
		{"малый бизнес"}, {"нефть"}, {"ритейл"}, {"ритейл", "крупный"}, {"металл"},
		{"нефть"}, {"малый бизнес"}, {"газ"}, {"нефть"}, {"газ", "крупный"},
		{"нефть"}, {"малый бизнес"}, {"металл"}, {"металл"}, {"золото"},
		{"нефть", "крупный"}, {"малый бизнес"}, {"металл"}, {"металл"}, {"металл"},
	}
	legalAddrJSON, _ := json.Marshal(map[string]string{"country": "Россия", "city": "Москва", "street": "ул. Примерная", "building": "1", "apartment": ""})
	actualAddrJSON, _ := json.Marshal(map[string]string{"country": "Россия", "city": "Москва", "street": "ул. Фактическая", "building": "2", "apartment": "5"})
	companyIDs := make([]uuid.UUID, 30)
	for i := 0; i < 30; i++ {
		companyIDs[i] = uuid.New()
		inn := fmt.Sprintf("77%02d%06d", (i%90)+10, (i*12345)%1000000)
		kpp := fmt.Sprintf("77%02d%05d", (i%90)+10, (i*54321)%100000)
		ogrn := fmt.Sprintf("1%02d%011d", (i%90)+10, (i*12345678901)%100000000000)
		phone := fmt.Sprintf("+7 (495) %03d-%02d-%02d", 100+i, (i*7)%100, (i*13)%100)
		email := fmt.Sprintf("info@company%d.ru", i+1)
		website := fmt.Sprintf("https://company%d.ru", i+1)
		_, err = db.ExecContext(ctx, `INSERT INTO crm_companies (id, workspace_id, name, inn, kpp, ogrn, phone, email, website, legal_address, actual_address, tags, owner_id, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10::jsonb,$11::jsonb,$12,$13,NOW(),NOW())`,
			companyIDs[i], wsID, companyNames[i], inn, kpp, ogrn, phone, email, website, string(legalAddrJSON), string(actualAddrJSON), pq.Array(companyTags[i]), uID)
		if err != nil {
			log.Fatalf("Insert company %d: %v", i+1, err)
		}
	}

	// 3. Contacts (100): все поля — middle_name, birthday, tags, custom_fields; несколько телефонов/email
	firstNames := []string{"Иван", "Мария", "Алексей", "Елена", "Дмитрий", "Анна", "Сергей", "Ольга", "Андрей", "Наталья",
		"Михаил", "Екатерина", "Павел", "Ирина", "Николай", "Татьяна", "Владимир", "Светлана", "Евгений", "Юлия",
		"Александр", "Виктория", "Максим", "Полина", "Артём", "Дарья", "Илья", "Кристина", "Кирилл", "Алина",
		"Роман", "Валерия", "Денис", "Анастасия", "Никита", "Елизавета", "Глеб", "Марина", "Константин", "Ксения",
		"Тимофей", "Вероника", "Арсений", "Ульяна", "Даниил", "София", "Матвей", "Арина", "Фёдор", "Василиса"}
	lastNames := []string{"Иванов", "Петров", "Сидоров", "Козлов", "Смирнов", "Кузнецов", "Попов", "Васильев", "Михайлов", "Новиков",
		"Федоров", "Морозов", "Волков", "Алексеев", "Лебедев", "Семенов", "Егоров", "Павлов", "Козлов", "Степанов",
		"Николаев", "Орлов", "Андреев", "Макаров", "Никитин", "Захаров", "Зайцев", "Соловьев", "Борисов", "Яковлев",
		"Григорьев", "Романов", "Воробьев", "Сергеев", "Кузьмин", "Фролов", "Александров", "Дмитриев", "Королев", "Гусев",
		"Киселев", "Ильин", "Максимов", "Поляков", "Сорокин", "Виноградов", "Ковалев", "Белов", "Медведев", "Антонов"}
	middleNames := []string{"Петрович", "Ивановна", "Сергеевич", "", "Александрович", "Дмитриевна", "", "Сергеевна", "Викторович", ""}
	positions := []string{"Директор", "Менеджер", "Коммерческий директор", "Руководитель отдела", "", "Бухгалтер", "ИТ-директор", "Маркетолог"}
	contactTagLists := [][]string{{"важный"}, {"лид"}, {"vip"}, {"повторный"}, {}, {"холодный"}, {"теплый"}, {}, {"ключевой"}, {}}
	now := time.Now().Format(time.RFC3339)
	contactIDs := make([]uuid.UUID, 100)
	contactToCompany := make(map[int]int)
	for i := 0; i < 70; i++ {
		contactToCompany[i] = i % 30
	}
	for i := 70; i < 100; i++ {
		contactToCompany[i] = -1
	}
	for i := 0; i < 100; i++ {
		contactIDs[i] = uuid.New()
		firstName := firstNames[i%50]
		lastName := lastNames[i%50]
		if i >= 50 {
			lastName = lastNames[(i/2)%50]
		}
		middleName := middleNames[i%len(middleNames)]
		var cid, midName, birthday interface{}
		pos := positions[i%len(positions)]
		if j := contactToCompany[i]; j >= 0 {
			cid = companyIDs[j]
		}
		if middleName != "" {
			midName = middleName
		}
		// День рождения для части контактов (формат DATE)
		if i%4 == 0 {
			birthday = fmt.Sprintf("19%02d-%02d-%02d", (80+i)%100, (i%12)+1, (i%28)+1)
		}
		tags := contactTagLists[i%len(contactTagLists)]
		var customFields interface{}
		if i%5 == 0 {
			b, _ := json.Marshal(map[string]interface{}{"source": "сайт", "comment": "интересный лид"})
			customFields = string(b)
		} else if i%7 == 0 {
			b, _ := json.Marshal(map[string]interface{}{"referrer": "партнёр"})
			customFields = string(b)
		}
		_, err = db.ExecContext(ctx, `INSERT INTO crm_contacts (id, workspace_id, first_name, last_name, middle_name, company_id, position, birthday, tags, owner_id, created_by, updated_by, created_at, updated_at, custom_fields) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$13,COALESCE($14::jsonb,'{}'::jsonb))`,
			contactIDs[i], wsID, firstName, lastName, midName, cid, pos, birthday, pq.Array(tags), uID, uID, uID, now, customFields)
		if err != nil {
			log.Fatalf("Insert contact %d: %v", i+1, err)
		}
		// Телефоны: mobile (основной) + work для части
		_, _ = db.ExecContext(ctx, `INSERT INTO crm_contact_phones (id, contact_id, type, number, is_primary) VALUES (gen_random_uuid(),$1,'mobile',$2,true)`,
			contactIDs[i], fmt.Sprintf("+7 9%02d %03d-%02d-%02d", (i%90)+10, (i*111)%1000, (i*17)%100, (i*23)%100))
		if i%3 == 0 {
			_, _ = db.ExecContext(ctx, `INSERT INTO crm_contact_phones (id, contact_id, type, number, is_primary) VALUES (gen_random_uuid(),$1,'work',$2,false)`,
				contactIDs[i], fmt.Sprintf("+7 (495) %03d-%02d-%02d", 100+i, (i*7)%100, (i*13)%100))
		}
		// Email: work (основной) + personal для части
		_, _ = db.ExecContext(ctx, `INSERT INTO crm_contact_emails (id, contact_id, type, address, is_primary) VALUES (gen_random_uuid(),$1,'work',$2,true)`,
			contactIDs[i], fmt.Sprintf("contact%d@company%d.ru", i+1, (i%30)+1))
		if i%4 == 0 {
			_, _ = db.ExecContext(ctx, `INSERT INTO crm_contact_emails (id, contact_id, type, address, is_primary) VALUES (gen_random_uuid(),$1,'personal',$2,false)`,
				contactIDs[i], fmt.Sprintf("person%d@gmail.com", i+1))
		}
	}
	// Company–contact links: для контактов 0..69 привязка к компании
	for i := 0; i < 70; i++ {
		coIdx := contactToCompany[i]
		pos := positions[i%len(positions)]
		_, _ = db.ExecContext(ctx, `INSERT INTO crm_company_contacts (company_id, contact_id, position, created_at) VALUES ($1,$2,$3,NOW())`,
			companyIDs[coIdx], contactIDs[i], pos)
	}

	// 4. Deals (20): все поля, распределены по 4 воронкам — expected_close_date, actual_close_date, lost_reason, description, source, probability, tags, currency (RUB/USD/EUR)
	dealNames := []string{
		"Поставка оборудования — Ромашка", "Консультация ИП Иванов", "Лицензии ПО — Северсталь",
		"Интеграция с 1С — Технопарк", "Аудит безопасности", "Подписка SaaS — год", "Разработка мобильного приложения",
		"Обучение сотрудников", "Техподдержка годовая", "Партнёрский договор — дилер", "Внедрение CRM",
		"Дизайн лендинга", "Рекламная кампания", "Подписка на облако", "Доработка сайта",
		"Консультация по налогам", "B2B: Корпоративная лицензия", "Ремонт офиса", "Охрана объекта", "Клининг",
	}
	dealContactIdx := []int{0, 71, 0, 3, 1, 72, 4, 0, 1, 5, 0, 73, 6, 74, 7, 75, 8, 2, 9, 76}
	dealCompanyIdx := []int{0, 1, 2, 3, 0, -1, 4, 0, 0, 2, 0, -1, 5, -1, 6, -1, 7, 1, 8, -1}
	dealPipelineIdx := []int{0, 0, 0, 0, 0, 1, 0, 0, 0, 2, 0, 1, 0, 1, 0, 0, 0, 3, 0, 0} // воронка 0..3
	dealStageIdx := []int{2, 0, 3, 2, 2, 2, 1, 2, 3, 1, 3, 0, 1, 0, 2, 0, 1, 1, 3, 2}    // стадия внутри воронки
	dealBudget := []float64{1500000, 50000, 1200000, 350000, 180000, 24000, 2500000, 85000, 120000, 890000, 1200000, 80000, 200000, 15000, 400000, 60000, 300000, 500000, 100000, 45000}
	dealCurrency := []string{"RUB", "RUB", "RUB", "RUB", "RUB", "USD", "RUB", "RUB", "EUR", "RUB", "RUB", "USD", "RUB", "RUB", "RUB", "RUB", "RUB", "RUB", "RUB", "RUB"}
	dealStatus := []string{"open", "open", "open", "open", "open", "won", "open", "open", "open", "open", "open", "open", "open", "won", "open", "open", "open", "open", "lost", "open"}
	dealSource := []string{"Сайт", "Звонок", "Рекомендация", "Сайт", "Звонок", "Сайт", "Сайт", "Рекомендация", "Выставка", "Партнёр", "Сайт", "Звонок", "Сайт", "Сайт", "Рекомендация", "Звонок", "Сайт", "Сайт", "Звонок", "Рекомендация"}
	dealTags := [][]string{
		{"крупный", "оборудование"}, {"консультация"}, {"важный", "повторный"}, {"интеграция"}, {"срочно"},
		{"подписка"}, {"крупный", "мобильное"}, {"обучение"}, {"подписка"}, {"партнёр"},
		{"крупный", "crm"}, {"дизайн"}, {"реклама"}, {"подписка"}, {"доработка"},
		{"консультация"}, {"b2b", "крупный"}, {"ремонт"}, {"охрана"}, {"клининг"},
	}
	winStageByPipe := []int{4, 3, 3, 4}  // индекс стадии "выигрыш" по воронкам (Сделка, Продлён, Выплата, Закрыто)
	lostStageByPipe := []int{5, 4, 4, 5} // индекс стадии "проигрыш"
	nowDate := time.Now().Format("2006-01-02")
	dealIDs := make([]uuid.UUID, 20)
	for i := 0; i < 20; i++ {
		dealIDs[i] = uuid.New()
		pIdx := dealPipelineIdx[i]
		sIdx := dealStageIdx[i]
		if dealStatus[i] == "won" {
			sIdx = winStageByPipe[pIdx]
		} else if dealStatus[i] == "lost" {
			sIdx = lostStageByPipe[pIdx]
		}
		var cid, coid interface{}
		if dealContactIdx[i] >= 0 {
			cid = contactIDs[dealContactIdx[i]]
		}
		if dealCompanyIdx[i] >= 0 {
			coid = companyIDs[dealCompanyIdx[i]]
		}
		var expectedClose, actualClose, lostReason, description interface{}
		expectedClose = nowDate
		if dealStatus[i] == "won" || dealStatus[i] == "lost" {
			actualClose = nowDate
		}
		if dealStatus[i] == "lost" {
			lostReason = "Выбрали конкурента"
		}
		description = "Тестовое описание сделки. Обсудили условия и сроки."
		prob := 10 + (sIdx * 20)
		if prob > 100 {
			prob = 100
		}
		_, err = db.ExecContext(ctx, `INSERT INTO crm_deals (id, workspace_id, name, contact_id, company_id, budget, currency, pipeline_id, stage_id, expected_close_date, actual_close_date, status, lost_reason, description, source, probability, tags, owner_id, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,NOW(),NOW())`,
			dealIDs[i], wsID, dealNames[i], cid, coid, dealBudget[i], dealCurrency[i], pipeIDs[pIdx], allStageIDs[pIdx][sIdx], expectedClose, actualClose, dealStatus[i], lostReason, description, dealSource[i], prob, pq.Array(dealTags[i]), uID)
		if err != nil {
			log.Fatalf("Insert deal %d: %v", i+1, err)
		}
	}

	// 5. Activity feed — записи для ленты активностей (выборка по сделкам, контактам, компаниям)
	userName := "Тестовый пользователь"
	optStr := func(s string) interface{} {
		if s == "" {
			return nil
		}
		return s
	}
	insertAct := func(entityType string, entityID uuid.UUID, actType, title, description string, isImportant, isEditable, isDeletable bool, metadata string) {
		if metadata == "" {
			metadata = "{}"
		}
		_, e := db.ExecContext(ctx, `INSERT INTO crm_activities (id, workspace_id, type, entity_type, entity_id, title, description, metadata, is_important, created_by, created_by_name, is_editable, is_deletable, created_at, updated_at) VALUES (gen_random_uuid(),$1,$2,$3,$4,$5,$6,$7::jsonb,$8,$9,$10,$11,$12,NOW(),NOW())`,
			wsID, actType, entityType, entityID, title, optStr(description), metadata, isImportant, uID, userName, isEditable, isDeletable)
		if e != nil {
			log.Fatalf("Insert activity %s %s: %v", actType, title, e)
		}
	}
	for i := 0; i < 5; i++ {
		insertAct("deal", dealIDs[i], "note", "Первичный контакт / КП", "Договорились о встрече или отправили КП.", false, true, true, "")
		insertAct("deal", dealIDs[i], "call", "Звонок", "", i == 0, false, false, `{"callDuration":120,"callDirection":"in","callStatus":"answered"}`)
	}
	for i := 0; i < 10; i++ {
		insertAct("contact", contactIDs[i], "contact_created", "Создание контакта", "", false, false, false, "")
		if i%2 == 0 {
			insertAct("contact", contactIDs[i], "note", "Заметка по контакту", "Обсудили условия.", false, true, true, "")
		}
	}
	for i := 0; i < 5; i++ {
		insertAct("company", companyIDs[i], "note", "Запросили КП", "Отправили коммерческое предложение.", false, true, true, "")
	}

	log.Printf("CRM seed done for workspace %s, user %s. Created: 4 pipelines, 30 companies, 100 contacts, 20 deals (all fields, all pipelines), activity feed (sample).", workspaceID, userID)
}
