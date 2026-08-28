-- 000003: 拓宽 dao_agents.avatar VARCHAR(255) → TEXT
-- 原因: 头像契约支持 data:image/(png|jpeg|webp|gif);base64 数据 URI(上限 1.5M 字符),
--       多数真实 data URI 超过 255 字符,旧列宽会导致保存报 value too long / 隐式截断,
--       形成「代码声称支持但数据库存不了」的混合状态。
ALTER TABLE dao_agents ALTER COLUMN avatar TYPE TEXT;
