DROP TABLE IF EXISTS notes;
DROP TABLE IF EXISTS counterparties;
DROP TABLE IF EXISTS currencies;

DELETE FROM modules WHERE code IN ('notes', 'inventory', 'finance', 'hr');
