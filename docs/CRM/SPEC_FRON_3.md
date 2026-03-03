## СПЕЦИФИКАЦИЯ ДЛЯ ФРОНТЕНДА
### Этап 3: Модуль TASKS (Задачи и напоминания)

**Отдельный независимый модуль**

---

## 1. Общая информация

**Цель:** Реализовать полноценный модуль задач, который может работать как отдельный продукт и интегрироваться с CRM.

**Где будет использоваться:**
- Отдельная страница "Мои задачи" (главная точка входа)
- Вкладка "Задачи" в карточках CRM (контакты, компании, сделки)
- Виджет задач в дашборде
- Уведомления о задачах

---

## 2. Типы данных (модели)

### 2.1. Task (Задача)

```typescript
export type TaskPriority = 'low' | 'medium' | 'high' | 'critical'
export type TaskStatus = 'pending' | 'in_progress' | 'completed' | 'cancelled'
export type TaskType = 'call' | 'meeting' | 'email' | 'lunch' | 'other'

export interface Task {
  id: string
  workspaceId: string
  
  // Основные поля
  title: string
  description?: string
  type: TaskType
  priority: TaskPriority
  status: TaskStatus
  
  // Даты
  dueDate: string  // ISO (когда нужно сделать)
  dueTime?: string // ЧЧ:ММ (опционально)
  reminderDate?: string // ISO (когда напомнить)
  completedAt?: string // ISO (когда выполнена)
  
  // Длительность
  duration?: number // в минутах
  
  // Исполнители
  assigneeId: string
  assigneeName?: string  // денормализовано
  assigneeAvatar?: string
  
  createdById: string
  createdByName: string
  createdByAvatar?: string
  
  completedById?: string
  completedByName?: string
  completionNote?: string
  
  // Повторение
  isRecurring: boolean
  recurringPattern?: RecurringPattern
  
  // Иерархия (подзадачи)
  parentId?: string
  subtasks?: Task[] // для UI
  
  // Связи с внешними сущностями (мягкие ссылки)
  entities: TaskEntityLink[]
  
  // Теги
  tags: Tag[]
  
  // Метаданные
  commentsCount: number
  isOverdue: boolean // вычисляемое на фронте
  isToday: boolean // вычисляемое на фронте
  
  // Системные
  createdAt: string
  updatedAt: string
}

export interface TaskEntityLink {
  type: 'crm_contact' | 'crm_deal' | 'crm_company'
  id: string
  name?: string // денормализовано для отображения
  url?: string // для перехода
}

export interface RecurringPattern {
  frequency: 'daily' | 'weekly' | 'monthly' | 'yearly'
  interval: number // каждые N дней/недель и т.д.
  weekdays?: number[] // для weekly: дни недели (1-7)
  monthday?: number // для monthly: число месяца
  endType: 'never' | 'date' | 'count'
  endDate?: string // для endType = 'date'
  endCount?: number // для endType = 'count'
  completedDates?: string[] // даты, когда уже выполнена
}

export interface Tag {
  id: string
  name: string
  color?: string
}

export interface TaskComment {
  id: string
  taskId: string
  comment: string
  createdBy: {
    id: string
    name: string
    avatar?: string
  }
  createdAt: string
  updatedAt: string
}
```

### 2.2. DTO для создания/обновления

```typescript
export interface CreateTaskDto {
  title: string
  description?: string
  type: TaskType
  priority: TaskPriority
  dueDate: string
  dueTime?: string
  reminderDate?: string
  duration?: number
  assigneeId: string
  isRecurring?: boolean
  recurringPattern?: RecurringPattern
  parentId?: string
  entities?: Array<{
    type: 'crm_contact' | 'crm_deal' | 'crm_company'
    id: string
  }>
  tags?: string[] // ID тегов
}

export interface UpdateTaskDto extends Partial<CreateTaskDto> {
  status?: TaskStatus
  completionNote?: string
}

export interface CompleteTaskDto {
  note?: string
  completionDate?: string // если нужно указать другую дату
}
```

### 2.3. Фильтры

```typescript
export interface TaskFilters {
  status?: TaskStatus[]
  priority?: TaskPriority[]
  type?: TaskType[]
  assigneeId?: string
  entityType?: 'crm_contact' | 'crm_deal' | 'crm_company'
  entityId?: string
  dueDateFrom?: string
  dueDateTo?: string
  overdue?: boolean
  today?: boolean
  upcoming?: boolean // ближайшие 7 дней
  search?: string
  tags?: string[]
  page?: number
  limit?: number
  sortBy?: 'dueDate' | 'priority' | 'createdAt' | 'title'
  sortOrder?: 'asc' | 'desc'
}

export interface TasksResponse {
  data: Task[]
  total: number
  page: number
  limit: number
  groups?: TaskGroup[] // для группированного отображения
}

export interface TaskGroup {
  id: string
  title: string // "Просрочено", "Сегодня", "Завтра", "На неделе", "Позже"
  tasks: Task[]
  count: number
}
```

### 2.4. Статистика

```typescript
export interface TasksStats {
  total: number
  byStatus: Record<TaskStatus, number>
  byPriority: Record<TaskPriority, number>
  overdue: number
  today: number
  upcoming: number // на 7 дней
  byAssignee: Array<{
    userId: string
    userName: string
    count: number
  }>
}
```

---

## 3. API Эндпоинты (ожидаемые от бекенда)

```
// Основные операции
GET    /api/v1/workspaces/{workspaceId}/tasks
GET    /api/v1/workspaces/{workspaceId}/tasks/{id}
POST   /api/v1/workspaces/{workspaceId}/tasks
PATCH  /api/v1/workspaces/{workspaceId}/tasks/{id}
DELETE /api/v1/workspaces/{workspaceId}/tasks/{id}

// Специальные
POST   /api/v1/workspaces/{workspaceId}/tasks/{id}/complete
POST   /api/v1/workspaces/{workspaceId}/tasks/{id}/reopen
POST   /api/v1/workspaces/{workspaceId}/tasks/{id}/remind

// Мои задачи
GET    /api/v1/workspaces/{workspaceId}/tasks/my
GET    /api/v1/workspaces/{workspaceId}/tasks/overdue
GET    /api/v1/workspaces/{workspaceId}/tasks/today

// Теги
GET    /api/v1/workspaces/{workspaceId}/tags
POST   /api/v1/workspaces/{workspaceId}/tags
DELETE /api/v1/workspaces/{workspaceId}/tags/{id}

// Комментарии
GET    /api/v1/workspaces/{workspaceId}/tasks/{taskId}/comments
POST   /api/v1/workspaces/{workspaceId}/tasks/{taskId}/comments
DELETE /api/v1/workspaces/{workspaceId}/tasks/{taskId}/comments/{id}

// Статистика
GET    /api/v1/workspaces/{workspaceId}/tasks/stats
```

---

## 4. Страницы и компоненты

### 4.1. MyTasksPage.vue (главная страница задач)

**URL:** `/tasks`

**Назначение:** Централизованный просмотр всех задач пользователя.

**Структура:**

```
┌─────────────────────────────────────────────────────────┐
│ ЗАДАЧИ                                      [➕ Создать] │
├─────────────────────────────────────────────────────────┤
│ ┌──────────────┬────────────────────────────────────┐   │
│ │ ФИЛЬТРЫ      │ СПИСОК ЗАДАЧ                        │   │
│ │              │                                      │   │
│ │ Статус:      │ 🔴 ПРОСРОЧЕНО (3)                    │   │
│ │ ☑ Все        │ ┌────────────────────────────────┐   │   │
│ │ ☐ В работе   │ │ 📞 14:30 Позвонить Иванову    │   │   │
│ │ ☐ Готово     │ │ 🔴 Высокий • Контакт          │   │   │
│ │              │ │ Сделка: Поставка             │   │   │
│ │ Приоритет:   │ └────────────────────────────────┘   │   │
│ │ ☑ Все        │                                      │   │
│ │ ☐ Высокий    │ 🟡 СЕГОДНЯ (5)                        │   │
│ │ ☐ Средний    │ ┌────────────────────────────────┐   │   │
│ │              │ │ ✉️ 10:00 Отправить КП         │   │   │
│ │ Тип:         │ │ 🟢 Средний • Компания         │   │   │
│ │ ☑ Все        │ └────────────────────────────────┘   │   │
│ │ ☐ Звонок     │                                      │   │
│ │ ☐ Встреча    │ 🟢 ЗАВТРА (2)                        │   │
│ │              │ ┌────────────────────────────────┐   │   │
│ │ [Применить]  │ │ 📅 09:00 Встреча с клиентом   │   │   │
│ │ [Сбросить]   │ │ 🔵 Низкий                      │   │   │
│ └──────────────┴────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

**Функционал:**
- Группировка задач по датам (просрочено, сегодня, завтра, на неделе, позже)
- Счетчики в группах
- Фильтры (статус, приоритет, тип, теги)
- Поиск по названию
- Кнопка создания задачи
- Переключение между видами: список / доска (опционально)

---

### 4.2. TaskCard.vue (карточка задачи в списке)

**Структура:**

```
┌────────────────────────────────────────────────────┐
│ ☐ [📞] 14:30 Позвонить Иванову              🔴 ⭐ ⋮ │
│     Контакт: Иван Петров • Сделка: Поставка       │
│     Исполнитель: Анна М • Создал: Павел 2ч назад  │
└────────────────────────────────────────────────────┘
```

**Элементы:**
- Чекбокс для отметки выполнения
- Иконка типа задачи (📞 звонок, 🤝 встреча, ✉️ письмо, 🍽️ обед, 📋 другое)
- Время (если есть)
- Заголовок
- Индикатор приоритета (цвет)
- Индикатор просрочки (🔴)
- Звездочка (важное)
- Кнопка меню (⋮)
- Связанные сущности (контакт, сделка, компания)
- Исполнитель и создатель

---

### 4.3. TaskDetailPage.vue (детальная карточка задачи)

**URL:** `/tasks/:id`

**Структура:**

```
┌────────────────────────────────────────────────────┐
│ ЗАДАЧА                                    [✎] [⋮]  │
├────────────────────────────────────────────────────┤
│                                                    │
│ [📞] Позвонить Иванову                              │
│                                                    │
│ Статус: ● В работе                                 │
│ Приоритет: 🔴 Высокий                              │
│ Тип: Звонок                                        │
│                                                    │
│ СРОКИ:                                             │
│ ┌────────────────────────────────────────────────┐ │
│ │ Выполнить: 15 марта 2026, 14:30                │ │
│ │ Напомнить: 15 марта 2026, 13:30 (за 1 час)     │ │
│ │ Длительность: 30 мин                            │ │
│ └────────────────────────────────────────────────┘ │
│                                                    │
│ СВЯЗИ:                                             │
│ ┌────────────────────────────────────────────────┐ │
│ │ 👤 Контакт: Иван Петров →                      │ │
│ │ 🏢 Компания: ООО Ромашка →                     │ │
│ │ 💼 Сделка: Поставка оборудования →             │ │
│ └────────────────────────────────────────────────┘ │
│                                                    │
│ ОПИСАНИЕ:                                          │
│ ┌────────────────────────────────────────────────┐ │
│ │ Обсудить условия поставки и скидки при объеме  │ │
│ │ от 100 единиц. Подготовить коммерческое        │ │
│ │ предложение.                                   │ │
│ └────────────────────────────────────────────────┘ │
│                                                    │
│ ТЕГИ: [важно] [срочно] [клиент]                   │
│                                                    │
│ ИСПОЛНИТЕЛЬ: Анна Менеджер                        │
│ СОЗДАЛ: Павел С. 2 дня назад                       │
│                                                    │
│ КОММЕНТАРИИ (3):                                  │
│ ┌────────────────────────────────────────────────┐ │
│ │ Анна: Обсудили, ждем КП                         │ │
│ │ Павел: Отправил КП 14.03                       │ │
│ └────────────────────────────────────────────────┘ │
│ [Добавить комментарий...]                          │
│                                                    │
│ [Выполнено] [Отложить] [Удалить]                  │
└────────────────────────────────────────────────────┘
```

**Вкладки (если нужно):**
- Детали
- Комментарии
- История изменений
- Подзадачи

---

### 4.4. TaskFormModal.vue (создание/редактирование)

**Структура:**

```
┌────────────────────────────────────────────────────┐
│ НОВАЯ ЗАДАЧА                                       │
├────────────────────────────────────────────────────┤
│                                                    │
│ Название *                                        │
│ [Позвонить Иванову.............................]  │
│                                                    │
│ Описание                                          │
│ [Обсудить условия поставки....................]  │
│ [............................................]  │
│                                                    │
│ Тип задачи                                        │
│ (📞) Звонок  (🤝) Встреча  (✉️) Письмо  (🍽️) Обед  │
│                                                    │
│ Приоритет                                         │
│ ○ Низкий ○ Средний ● Высокий ○ Критичный         │
│                                                    │
│ Срок выполнения *                                 │
│ [15.03.2026] [14:30]                               │
│                                                    │
│ Напоминание                                       │
│ ☑ Напомнить за [1 час ▼]                          │
│                                                    │
│ Длительность                                      │
│ [30] минут                                        │
│                                                    │
│ СВЯЗАТЬ С CRM (опционально):                      │
│ Контакт: [Иван Петров ▼]                           │
│ Компания: [ООО Ромашка ▼]                          │
│ Сделка: [Поставка ▼]                              │
│                                                    │
│ Исполнитель *                                     │
│ [Анна Менеджер ▼]                                 │
│                                                    │
│ Теги                                              │
│ [важно] [срочно] [ + Добавить]                    │
│                                                    │
│ ПОВТОРЕНИЕ                                        │
│ ☐ Повторять каждый [день ▼]                       │
│   ○ Бесконечно  ○ До [31.12.2026]  ○ [10] раз    │
│                                                    │
│ ПОДЗАДАЧИ (опционально):                          │
│ [ + Добавить подзадачу ]                          │
│                                                    │
│ [Отмена]              [Сохранить и еще] [Сохранить]│
└────────────────────────────────────────────────────┘
```

**Валидация:**
- Название — обязательно
- Срок выполнения — обязательно, не может быть в прошлом
- Исполнитель — обязательно
- Если выбрана связь с CRM, проверить существование

---

### 4.5. TaskQuickAdd.vue (быстрое добавление)

**Используется:** В дашборде, в карточках CRM

**Структура:**

```
┌────────────────────────────────────────────────────┐
│ 📝 + Добавить задачу...                            │
└────────────────────────────────────────────────────┘

(при клике раскрывается)

┌────────────────────────────────────────────────────┐
│ Название: [Позвонить клиенту..................]   │
│ Срок:    [15.03] [14:30]                          │
│ Исполнитель: [Анна ▼]                             │
│ Связать: [Контакт ▼] [Выбрать...]                 │
│                                                   │
│ [Отмена] [Быстрое] [Создать]                      │
└────────────────────────────────────────────────────┘
```

---

### 4.6. TaskFilters.vue (панель фильтров)

**Структура:**

```
┌────────────────────────────────────────────────────┐
│ ФИЛЬТРЫ ЗАДАЧ                                      │
├────────────────────────────────────────────────────┤
│                                                    │
│ СТАТУС                                             │
│ ☑ Все                                              │
│ ☐ В работе                                         │
│ ☐ Выполненные                                      │
│ ☐ Отмененные                                       │
│                                                    │
│ ПРИОРИТЕТ                                          │
│ ☑ Все                                              │
│ ☐ Критичный                                        │
│ ☐ Высокий                                          │
│ ☐ Средний                                          │
│ ☐ Низкий                                           │
│                                                    │
│ ТИП                                                │
│ ☑ Все                                              │
│ ☐ Звонки                                           │
│ ☐ Встречи                                          │
│ ☐ Письма                                           │
│ ☐ Обеды                                            │
│                                                    │
│ ПЕРИОД                                             │
│ ○ Все время                                        │
│ ○ Сегодня                                          │
│ ○ Завтра                                           │
│ ○ На этой неделе                                   │
│ ○ Произвольный                                     │
│   [с: 01.03.2026] [по: 15.03.2026]                 │
│                                                    │
│ ДОПОЛНИТЕЛЬНО                                      │
│ ☐ Только мои задачи                                │
│ ☐ Просроченные                                     │
│ ☐ Без исполнителя                                  │
│                                                    │
│ ТЕГИ                                               │
│ [важно] [срочно] [клиент] [ + ]                    │
│                                                    │
│ ПОИСК                                              │
│ [............................................]    │
│                                                    │
│ [Сбросить]                        [Применить]     │
└────────────────────────────────────────────────────┘
```

---

### 4.7. TaskComments.vue (комментарии)

**Структура:**

```
┌────────────────────────────────────────────────────┐
│ КОММЕНТАРИИ (3)                                    │
├────────────────────────────────────────────────────┤
│                                                    │
│ ┌────────────────────────────────────────────────┐ │
│ │ Анна Менеджер               15.03 14:30        │ │
│ │ Обсудили условия, ждем коммерческое            │ │
│ │ предложение                                    │ │
│ └────────────────────────────────────────────────┘ │
│                                                    │
│ ┌────────────────────────────────────────────────┐ │
│ │ Павел С.                     15.03 15:45       │ │
│ │ Отправил КП на почту. Ждем ответа              │ │
│ └────────────────────────────────────────────────┘ │
│                                                    │
│ [Напишите комментарий...]                [📎] [➤]  │
└────────────────────────────────────────────────────┘
```

---

### 4.8. TaskStatsWidget.vue (виджет статистики)

**Используется:** В дашборде

**Структура:**

```
┌────────────────────────────────────────────────────┐
│ ЗАДАЧИ                              [Все задачи →] │
├────────────────────────────────────────────────────┤
│                                                    │
│ ┌──────────┐  ┌──────────┐  ┌──────────┐          │
│ │ 12       │  │ 5        │  │ 3        │          │
│ │ Сегодня  │  │ Просроч. │  │ На неделе│          │
│ └──────────┘  └──────────┘  └──────────┘          │
│                                                    │
│ ┌────────────────────────────────────────────────┐ │
│ │ Прогресс за неделю                   ▰▰▰▰▰▱▱ 70%│ │
│ └────────────────────────────────────────────────┘ │
│                                                    │
│ БЛИЖАЙШИЕ:                                        │
│ • 14:30 Позвонить Иванову (сегодня)               │
│ • 10:00 Встреча (завтра)                          │
└────────────────────────────────────────────────────┘
```

---

## 5. Интеграция с CRM

### 5.1. В карточке контакта

```vue
<Tabs>
  <Tab title="Основная информация">...</Tab>
  <Tab title="Сделки">...</Tab>
  <Tab title="Активность">...</Tab>
  <Tab title="Задачи">
    <TaskList 
      :filters="{
        entityType: 'crm_contact',
        entityId: contactId
      }"
      :can-create="true"
      @create="openTaskFormWithContact"
    />
    <TaskQuickAdd 
      entity-type="crm_contact"
      :entity-id="contactId"
      :entity-name="contact.name"
    />
  </Tab>
</Tabs>
```

### 5.2. В карточке сделки

Аналогично, с `entityType: 'crm_deal'`

### 5.3. В карточке компании

Аналогично, с `entityType: 'crm_company'`

---

## 6. Состояния интерфейса

### 6.1. Загрузка

```
┌────────────────────────────────────────────────────┐
│ ██████████████████████████████████████████████████ │
│ ██████████████████████████████████████████████████ │
│ ██████████████████████████████████████████████████ │
└────────────────────────────────────────────────────┘
```

### 6.2. Пусто (нет задач)

```
┌────────────────────────────────────────────────────┐
│ 📭 У вас пока нет задач                             │
│                                                    │
│ Создайте первую задачу, чтобы начать планирование  │
│                                                    │
│ [➕ Создать задачу]                                 │
└────────────────────────────────────────────────────┘
```

### 6.3. Фильтр ничего не нашел

```
┌────────────────────────────────────────────────────┐
│ 🔍 Нет задач, соответствующих выбранным фильтрам   │
│                                                    │
│ [Сбросить фильтры]                                 │
└────────────────────────────────────────────────────┘
```

### 6.4. Ошибка

```
┌────────────────────────────────────────────────────┐
│ ⚠️ Не удалось загрузить задачи                      │
│                                                    │
│ [Повторить]                                        │
└────────────────────────────────────────────────────┘
```

---

## 7. Группировка и сортировка

### 7.1. Группы по умолчанию

```typescript
const TASK_GROUPS = [
  {
    id: 'overdue',
    title: 'Просрочено',
    filter: (task) => task.isOverdue && task.status !== 'completed',
    color: 'red'
  },
  {
    id: 'today',
    title: 'Сегодня',
    filter: (task) => task.isToday && task.status !== 'completed',
    color: 'yellow'
  },
  {
    id: 'tomorrow',
    title: 'Завтра',
    filter: (task) => isTomorrow(task.dueDate) && task.status !== 'completed',
    color: 'blue'
  },
  {
    id: 'week',
    title: 'На этой неделе',
    filter: (task) => isThisWeek(task.dueDate) && task.status !== 'completed',
    color: 'green'
  },
  {
    id: 'later',
    title: 'Позже',
    filter: (task) => !isThisWeek(task.dueDate) && task.status !== 'completed',
    color: 'gray'
  },
  {
    id: 'completed',
    title: 'Выполненные',
    filter: (task) => task.status === 'completed',
    color: 'gray'
  }
]
```

### 7.2. Сортировка внутри групп

```typescript
const sortTasks = (tasks: Task[]) => {
  return tasks.sort((a, b) => {
    // Сначала по приоритету
    const priorityOrder = { critical: 0, high: 1, medium: 2, low: 3 }
    if (a.priority !== b.priority) {
      return priorityOrder[a.priority] - priorityOrder[b.priority]
    }
    // Потом по дате
    return new Date(a.dueDate).getTime() - new Date(b.dueDate).getTime()
  })
}
```

---

## 8. Уведомления (Realtime)

### 8.1. Типы уведомлений

```typescript
interface TaskNotification {
  type: 'task_reminder' | 'task_assigned' | 'task_overdue' | 'task_completed'
  taskId: string
  taskTitle: string
  minutesUntilDue?: number
}
```

### 8.2. Визуал уведомлений

```
┌────────────────────────────────────────────────────┐
│ 🔔 Напоминание о задаче                             │
│ "Позвонить Иванову" через 15 минут                 │
│ [Открыть] [Отложить на 5 мин]                       │
└────────────────────────────────────────────────────┘
```

---

## 9. Store (состояние)

### 9.1. Task Store

```typescript
interface TaskState {
  tasks: Task[]
  currentTask: Task | null
  filters: TaskFilters
  groups: TaskGroup[]
  stats: TasksStats | null
  loading: boolean
  error: string | null
  pagination: {
    page: number
    limit: number
    total: number
  }
}

interface TaskActions {
  // Загрузка
  loadTasks(filters?: TaskFilters): Promise<void>
  loadMore(): Promise<void>
  loadTask(id: string): Promise<void>
  loadStats(): Promise<void>
  
  // CRUD
  createTask(data: CreateTaskDto): Promise<Task>
  updateTask(id: string, data: UpdateTaskDto): Promise<Task>
  deleteTask(id: string): Promise<void>
  
  // Операции
  completeTask(id: string, note?: string): Promise<Task>
  reopenTask(id: string): Promise<Task>
  setReminder(id: string, date: string): Promise<void>
  
  // Фильтры
  setFilters(filters: TaskFilters): void
  resetFilters(): void
  
  // Группировка
  groupTasks(): TaskGroup[]
}
```

---

## 10. Критерии готовности

### Функциональные
- [ ] Можно создать задачу со всеми полями
- [ ] Можно редактировать задачу
- [ ] Можно удалить задачу
- [ ] Можно отметить задачу выполненной
- [ ] Работают повторяющиеся задачи
- [ ] Работают подзадачи
- [ ] Работают комментарии
- [ ] Работают теги
- [ ] Работают все фильтры
- [ ] Работает поиск
- [ ] Работает группировка по датам
- [ ] Работает интеграция с CRM

### UI/UX
- [ ] Все состояния (загрузка, ошибка, пусто) корректны
- [ ] Индикаторы приоритетов видны
- [ ] Иконки типов понятны
- [ ] На мобильных устройствах адаптируется
- [ ] Уведомления приходят вовремя

### Интеграция
- [ ] В CRM видно задачи по контакту/сделке
- [ ] Из задачи можно перейти в CRM
- [ ] При создании задачи из CRM подставляются связи

---

## 11. Роутинг

```typescript
const routes = [
  {
    path: '/tasks',
    name: 'tasks',
    component: MyTasksPage,
    meta: { requiresWorkspace: true }
  },
  {
    path: '/tasks/:id',
    name: 'task-detail',
    component: TaskDetailPage,
    meta: { requiresWorkspace: true }
  },
  {
    path: '/tasks/calendar',
    name: 'task-calendar',
    component: TaskCalendarView,
    meta: { requiresWorkspace: true }
  },
  {
    path: '/tasks/board',
    name: 'task-board',
    component: TaskBoardView,
    meta: { requiresWorkspace: true }
  }
]

