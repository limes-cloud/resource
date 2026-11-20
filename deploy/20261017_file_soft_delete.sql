-- 文件软删除：file 增加 status（1正常 2待删除）+ 后台回收扫描索引。
-- 删目录改为「标记子树文件待删除 + 后台分批物理回收」，不再在请求里逐个删桶。

ALTER TABLE `file`
  ADD COLUMN `status` TINYINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '1正常 2待删除' AFTER `upload_id`,
  ADD KEY `idx_file_status` (`status`, `id`);
