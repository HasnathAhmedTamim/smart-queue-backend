PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS services (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  code TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS tokens (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  service_id INTEGER NOT NULL,
  token_code TEXT NOT NULL UNIQUE,
  customer_name TEXT,
  status TEXT NOT NULL CHECK(status IN ('waiting','serving','done')) DEFAULT 'waiting',
  created_at TEXT NOT NULL,
  served_at TEXT,
  done_at TEXT,
  FOREIGN KEY(service_id) REFERENCES services(id)
);

INSERT OR IGNORE INTO services(code, name) VALUES
('A','Account Opening'),
('D','Deposit'),
('L','Loan Desk');