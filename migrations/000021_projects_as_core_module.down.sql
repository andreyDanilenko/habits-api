-- Откат: проекты больше не считаются core-модулем.
-- Записи в workspace_modules не удаляем: воркспейсы, где проекты уже были включены, сохраняют доступ.

UPDATE modules SET is_core = FALSE WHERE code = 'projects';

