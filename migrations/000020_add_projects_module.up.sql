-- Модуль «Проекты»: группировка сущностей из CRM и других модулей в контексты (см. docs/PROJECTS_SPEC.md).
INSERT INTO modules (id, code, name, description, is_core) VALUES
    (gen_random_uuid(), 'projects', 'Проекты', 'Группировка контактов, компаний, сделок и других сущностей в проекты', FALSE);
