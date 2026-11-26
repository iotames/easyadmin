SELECT name FROM sqlite_master WHERE type = 'index' AND tbl_name = '{table}' AND name NOT LIKE 'sqlite_%';
