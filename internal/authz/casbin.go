package authz

import (
	casbin "github.com/casbin/casbin/v3"
	casbinModel "github.com/casbin/casbin/v3/model"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"gorm.io/gorm"
)

// InitEnforcer инициализирует Casbin с поддержкой доменов (workspace) и хранилищем политик в БД через GORM-адаптер.
func InitEnforcer(db *gorm.DB) (*casbin.Enforcer, error) {
	// Адаптер использует стандартную таблицу casbin_rule
	adapter, err := gormadapter.NewAdapterByDB(db)
	if err != nil {
		return nil, err
	}

	// Модель с доменами (workspace_id) и иерархией ролей через g (sub, parent, domain)
	const modelText = `
	[request_definition]
	r = sub, dom, obj, act

	[policy_definition]
	p = sub, dom, obj, act

	[role_definition]
	g = _, _, _

	[policy_effect]
	e = some(where (p.eft == allow))

	[matchers]
	m = g(r.sub, p.sub, r.dom) && r.dom == p.dom && r.obj == p.obj && r.act == p.act
	`

	m, err := casbinModel.NewModelFromString(modelText)
	if err != nil {
		return nil, err
	}

	e, err := casbin.NewEnforcer(m, adapter)
	if err != nil {
		return nil, err
	}

	// Загружаем политики из БД
	if err := e.LoadPolicy(); err != nil {
		return nil, err
	}

	return e, nil
}

