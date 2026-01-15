CREATE TABLE IF NOT EXISTS request_logs (
    id SERIAL PRIMARY KEY,
    timestamp TIMESTAMP NOT NULL,
    status_code INTEGER NOT NULL,
    duration_ms DECIMAL(10, 6) NOT NULL,
    client_ip VARCHAR(45) NOT NULL,
    method VARCHAR(10) NOT NULL,
    path TEXT NOT NULL,
    raw_log TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_request_logs_timestamp ON request_logs(timestamp);
