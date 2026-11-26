CREATE INDEX IF NOT EXISTS idx_sys_audit_log_created_at ON sys_audit_log(created_at);
CREATE INDEX IF NOT EXISTS idx_sys_audit_log_actor_id ON sys_audit_log(actor_id);
CREATE INDEX IF NOT EXISTS idx_sys_audit_log_action ON sys_audit_log(action);
