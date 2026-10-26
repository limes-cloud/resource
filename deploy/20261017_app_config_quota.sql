-- 应用存储配额：app_config 增加 quota / used_bytes 两列，并回填存量用量。
-- quota=0 表示不限；存量应用默认不限，由管理员按需设置上限（不批量套 50G）。

ALTER TABLE `app_config`
  ADD COLUMN `quota`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '存储配额(字节),0=不限' AFTER `export_ttl`,
  ADD COLUMN `used_bytes` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '已用量(字节)' AFTER `quota`;

-- 存量回填：按逻辑口径（该 app 每个 file 行引用 object 的 size 之和）重算 used_bytes。
UPDATE `app_config` a
   SET a.used_bytes = (
     SELECT COALESCE(SUM(o.size), 0)
       FROM `file` f
       JOIN `object` o ON o.id = f.object_id
      WHERE f.app = a.app
   );
