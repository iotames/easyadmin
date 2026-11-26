CREATE INDEX IF NOT EXISTS idx_sys_session_user_id ON sys_session(user_id);
CREATE INDEX IF NOT EXISTS idx_sys_session_expires_at ON sys_session(expires_at);
