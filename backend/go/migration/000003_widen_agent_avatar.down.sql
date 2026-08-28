-- 000003 down: 还原列宽。若已有超过 255 字符的 data URI 数据,此 ALTER 会报错,属预期(数据不兼容旧 schema)。
ALTER TABLE dao_agents ALTER COLUMN avatar TYPE VARCHAR(255);
