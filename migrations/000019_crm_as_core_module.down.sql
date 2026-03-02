-- Откат: CRM снова не core. Записи в workspace_modules не удаляем — воркспейсы сохраняют доступ к CRM.
UPDATE modules SET is_core = FALSE WHERE code = 'crm';
