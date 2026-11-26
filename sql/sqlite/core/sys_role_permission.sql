CREATE TABLE IF NOT EXISTS sys_role_permission (
  role_id INTEGER NOT NULL REFERENCES sys_role(id) ON DELETE CASCADE,
  permission_id INTEGER NOT NULL REFERENCES sys_permission(id) ON DELETE CASCADE,
  PRIMARY KEY(role_id, permission_id)
);
