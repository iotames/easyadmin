CREATE TABLE IF NOT EXISTS sys_audit_log (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  actor_id INTEGER NULL,
  actor_name TEXT NOT NULL DEFAULT '',
  action TEXT NOT NULL,
  outcome TEXT NOT NULL,
  target_type TEXT NOT NULL DEFAULT '',
  target_id TEXT NOT NULL DEFAULT '',
  summary TEXT NOT NULL,
  detail_json TEXT NULL,
  ip TEXT NOT NULL DEFAULT '',
  user_agent TEXT NOT NULL DEFAULT '',
  request_id TEXT NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL
);
