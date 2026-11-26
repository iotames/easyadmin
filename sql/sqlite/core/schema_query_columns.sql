SELECT
  ti.name AS name,
  ti.type AS type,
  CASE WHEN ti."notnull" = 1 THEN 'NO' ELSE 'YES' END AS nullable,
  ti.dflt_value AS dflt,
  CASE
    WHEN ti.pk > 0
      AND upper(ti.type) = 'INTEGER'
      AND (SELECT COUNT(*) FROM pragma_table_info('{table}') p WHERE p.pk > 0) = 1
      AND (SELECT upper(m.sql) FROM sqlite_master m WHERE m.type = 'table' AND m.name = '{table}') LIKE '%AUTOINCREMENT%'
    THEN 'auto_increment' ELSE '' END AS extra
FROM pragma_table_info('{table}') ti
ORDER BY ti.cid;
