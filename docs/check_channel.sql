-- 检查 APIMart 渠道（ID=14）的 models 和 group 字段
SELECT id, name, type, status, models, `group`, base_url FROM channels WHERE id = 14;

-- 检查 abilities 表中是否有 gpt-image-2 的记录
SELECT * FROM abilities WHERE channel_id = 14;

-- 检查 default 分组下所有可用的 gpt-image-2 渠道
SELECT a.*, c.name, c.type FROM abilities a
JOIN channels c ON a.channel_id = c.id
WHERE a.model = 'gpt-image-2' AND a.group = 'default' AND a.enabled = 1;

-- 如果 channels.models 字段缺少 gpt-image-2，手动修复：
-- UPDATE channels SET models = 'gpt-image-2,veo3.1-fast' WHERE id = 14;

-- 修复后运行 FixAbility 或执行以下 SQL 重建 abilities：
-- DELETE FROM abilities WHERE channel_id = 14;
-- INSERT INTO abilities (`group`, model, channel_id, enabled, priority, weight)
-- VALUES 
--   ('default', 'gpt-image-2', 14, 1, 0, 0),
--   ('default', 'veo3.1-fast', 14, 1, 0, 0);
