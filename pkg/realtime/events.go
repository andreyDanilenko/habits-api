package realtime

// Realtime event type constants.
// Use these instead of string literals to ensure consistency across services.
const (
	// Habits
	EventHabitCreated   = "habit.created"
	EventHabitUpdated   = "habit.updated"
	EventHabitDeleted   = "habit.deleted"
	EventHabitCompleted = "habit.completed"

	// Activity (habits/journal)
	EventActivityCreated = "activity.created"

	// Deals (CRM)
	EventDealCreated = "deal.created"
	EventDealUpdated = "deal.updated"
	EventDealDeleted = "deal.deleted"

	// Tasks
	EventTaskCreated   = "task.created"
	EventTaskUpdated   = "task.updated"
	EventTaskDeleted   = "task.deleted"
	EventTaskCompleted = "task.completed"
	EventTaskReopened  = "task.reopened"

	// Invitations
	EventInvitationAccepted = "invitation.accepted"

	// Chat
	EventChatMessageCreated  = "chat.message.created"
	EventChatThreadUpserted  = "chat.thread.upserted"
)
