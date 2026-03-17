-- Расширение типов задач: более универсальные (task, bug, feature, meeting, call, other)
-- Сохраняем обратную совместимость: call, meeting, other остаются
ALTER TABLE tasks DROP CONSTRAINT IF EXISTS tasks_type_check;
ALTER TABLE tasks ADD CONSTRAINT tasks_type_check CHECK (
  type IN ('task', 'bug', 'feature', 'meeting', 'call', 'email', 'lunch', 'other')
);
-- Миграция существующих: call, meeting, email, lunch, other -> остаются как есть
-- Новые значения: task (по умолчанию для новых), bug, feature
