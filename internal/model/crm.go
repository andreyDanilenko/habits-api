package model

// ContactPhone and ContactEmail for API JSON
type ContactPhone struct {
	Type      string `json:"type"`
	Number    string `json:"number"`
	IsPrimary bool   `json:"isPrimary"`
}

type ContactEmail struct {
	Type      string `json:"type"`
	Address   string `json:"address"`
	IsPrimary bool   `json:"isPrimary"`
}

type Contact struct {
	ID           string                 `json:"id"`
	FirstName    string                 `json:"firstName"`
	LastName     string                 `json:"lastName"`
	MiddleName   string                 `json:"middleName,omitempty"`
	Phones       []ContactPhone         `json:"phones"`
	Emails       []ContactEmail         `json:"emails"`
	CompanyID    string                 `json:"companyId,omitempty"`
	Position     string                 `json:"position,omitempty"`
	Birthday     string                 `json:"birthday,omitempty"`
	Tags         []string               `json:"tags"`
	OwnerID      string                 `json:"ownerId"`
	CreatedBy    string                 `json:"createdBy"`
	UpdatedBy    string                 `json:"updatedBy"`
	CreatedAt    string                 `json:"createdAt"`
	UpdatedAt    string                 `json:"updatedAt"`
	CustomFields map[string]interface{}  `json:"customFields,omitempty"`
}

type CompanyAddress struct {
	Country   string `json:"country"`
	City      string `json:"city"`
	Street    string `json:"street"`
	Building  string `json:"building"`
	Apartment string `json:"apartment,omitempty"`
}

type Company struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	INN            string          `json:"inn,omitempty"`
	KPP            string          `json:"kpp,omitempty"`
	OGRN           string          `json:"ogrn,omitempty"`
	Phone          string          `json:"phone,omitempty"`
	Email          string          `json:"email,omitempty"`
	Website        string          `json:"website,omitempty"`
	LegalAddress   *CompanyAddress `json:"legalAddress,omitempty"`
	ActualAddress  *CompanyAddress `json:"actualAddress,omitempty"`
	Contacts       []string        `json:"contacts"`
	Tags           []string        `json:"tags"`
	OwnerID        string          `json:"ownerId"`
	CreatedAt      string          `json:"createdAt"`
	UpdatedAt      string          `json:"updatedAt"`
}

type Stage struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Order      int    `json:"order"`
	Color      string `json:"color,omitempty"`
	Probability int   `json:"probability"`
	IsFinal    bool   `json:"isFinal"`
	IsLost     bool   `json:"isLost"`
}

type Pipeline struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Stages    []Stage `json:"stages"`
	IsDefault bool    `json:"isDefault"`
}

type Deal struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	ContactID         string   `json:"contactId,omitempty"`
	CompanyID         string   `json:"companyId,omitempty"`
	Budget            float64  `json:"budget"`
	Currency          string   `json:"currency"`
	PipelineID        string   `json:"pipelineId"`
	StageID           string   `json:"stageId"`
	ExpectedCloseDate string   `json:"expectedCloseDate,omitempty"`
	ActualCloseDate   string   `json:"actualCloseDate,omitempty"`
	Status            string   `json:"status"`
	LostReason        string   `json:"lostReason,omitempty"`
	Description       string   `json:"description,omitempty"`
	Source            string   `json:"source,omitempty"`
	Probability       *int     `json:"probability,omitempty"`
	Tags              []string `json:"tags"`
	OwnerID           string   `json:"ownerId"`
	CreatedAt         string   `json:"createdAt"`
	UpdatedAt         string   `json:"updatedAt"`
}

// CrmActivity (CRM Activity Feed, SPEC_BACK_2) — JSON as "Activity" for frontend
type CrmActivityCreatedBy struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar,omitempty"`
}

type CrmActivity struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	EntityType  string                 `json:"entityType"`
	EntityID    string                 `json:"entityId"`
	Title       string                 `json:"title"`
	Description string                 `json:"description,omitempty"`
	Metadata    map[string]interface{}  `json:"metadata,omitempty"`
	IsImportant bool                   `json:"isImportant,omitempty"`
	CreatedBy   CrmActivityCreatedBy   `json:"createdBy"`
	CreatedAt   string                 `json:"createdAt"`
	IsEditable  bool                   `json:"isEditable,omitempty"`
	IsDeletable bool                   `json:"isDeletable,omitempty"`
}
