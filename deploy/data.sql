
SET SQL_MODE = "NO_AUTO_VALUE_ON_ZERO";
START TRANSACTION;
SET time_zone = "+00:00";


/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8mb4 */;

--
-- 数据库： `resource`
--

-- --------------------------------------------------------

--
-- 表的结构 `chunk`
--
DROP TABLE IF EXISTS `chunk`;
CREATE TABLE `chunk` (
                         `id` bigint(20) NOT NULL COMMENT '分片id',
                         `upload_id` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT '上传id',
                         `index` bigint(20) NOT NULL COMMENT '切片下标',
                         `size` bigint(20) NOT NULL COMMENT '切片大小',
                         `sha` varchar(128) NOT NULL COMMENT '切片sha',
                         `data` mediumblob NOT NULL COMMENT '切片数据',
                         `created_at` bigint(20) UNSIGNED DEFAULT NULL COMMENT '创建时间'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- --------------------------------------------------------

--
-- 表的结构 `directory`
--
DROP TABLE IF EXISTS `directory`;
CREATE TABLE `directory` (
                             `id` bigint(20) UNSIGNED NOT NULL COMMENT '主键ID',
                             `parent_id` bigint(20) UNSIGNED NOT NULL COMMENT '父id',
                             `name` varchar(64) NOT NULL COMMENT '目录名称',
                             `accept` tinytext NOT NULL COMMENT '允许后缀',
                             `max_size` bigint(20) UNSIGNED NOT NULL COMMENT '最大大小',
                             `created_at` bigint(20) DEFAULT NULL COMMENT '创建时间',
                             `updated_at` bigint(20) DEFAULT NULL COMMENT '修改时间'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='目录信息';

--
-- 转存表中的数据 `directory`
--

INSERT INTO `directory` (`id`, `parent_id`, `name`, `accept`, `max_size`, `created_at`, `updated_at`) VALUES
                                                                                                          (1, 0, 'manager', 'jpg,png,txt,ppt,pptx,mp4,pdf', 10, 1717784031, 1717784031),
                                                                                                          (2, 1, 'avatar', 'jpg,png,txt,ppt,pptx,mp4,pdf', 10, 1717784031, 1718105158),
                                                                                                          (6, 0, 'channel', 'jpg,png,txt,ppt,pptx,mp4,pdf', 10, 1718391488, 1718391488),
                                                                                                          (7, 6, 'logo', 'jpg,png,txt,ppt,pptx,mp4,pdf', 10, 1718391488, 1718391488),
                                                                                                          (8, 0, 'usercenter', 'jpg,png,txt,ppt,pptx,mp4,pdf', 10, 1718438213, 1718438213),
                                                                                                          (9, 8, 'app', 'jpg,png,txt,ppt,pptx,mp4,pdf', 10, 1718438213, 1718438213),
                                                                                                          (10, 9, 'logo', 'jpg,png,txt,ppt,pptx,mp4,pdf', 10, 1718438213, 1718438213),
                                                                                                          (11, 0, 'user', 'jpg,png,txt,ppt,pptx,mp4,pdf', 10, 1719068487, 1719068487),
                                                                                                          (12, 11, 'logo', 'jpg,png,txt,ppt,pptx,mp4,pdf', 10, 1719068487, 1719068487);

-- --------------------------------------------------------

--
-- 表的结构 `directory_closure`
--
DROP TABLE IF EXISTS `directory_closure`;
CREATE TABLE `directory_closure` (
                                     `id` bigint(20) UNSIGNED NOT NULL COMMENT '主键ID',
                                     `parent` bigint(20) UNSIGNED NOT NULL COMMENT '目录id',
                                     `children` bigint(20) UNSIGNED NOT NULL COMMENT '目录id'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='目录层级信息';

-- --------------------------------------------------------

--
-- 表的结构 `export`
--
DROP TABLE IF EXISTS `export`;
CREATE TABLE `export` (
                          `id` bigint(20) NOT NULL COMMENT '主键ID',
                          `scene` char(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT '场景',
                          `name` varchar(128) NOT NULL COMMENT '名称',
                          `size` bigint(20) NOT NULL DEFAULT '0' COMMENT '大小',
                          `sha` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT '版本',
                          `src` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL COMMENT '路径',
                          `reason` varchar(512) DEFAULT NULL COMMENT '错误原因',
                          `status` varchar(32) NOT NULL DEFAULT '' COMMENT '状态',
                          `user_id` bigint(20) UNSIGNED DEFAULT NULL COMMENT '创建人',
                          `department_id` bigint(20) UNSIGNED DEFAULT NULL COMMENT '创建部门',
                          `expired_at` bigint(20) UNSIGNED DEFAULT NULL COMMENT '过期时间',
                          `created_at` bigint(20) UNSIGNED DEFAULT NULL COMMENT '创建时间',
                          `updated_at` bigint(20) UNSIGNED DEFAULT NULL COMMENT '修改时间'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='导出任务';

--
-- 转存表中的数据 `export`
--

INSERT INTO `export` (`id`, `scene`, `name`, `size`, `sha`, `src`, `reason`, `status`, `user_id`, `department_id`, `expired_at`, `created_at`, `updated_at`) VALUES
    (8, 'ResourceExport', '测试导出', 0, '37a6259cc0c1dae299a7866489dff0bd', '37a6259cc0c1dae299a7866489dff0bd.zip', NULL, 'COMPLETED', 1, 1, 1719726031, 1719466831, 1719466831);

-- --------------------------------------------------------

--
-- 表的结构 `file`
--
DROP TABLE IF EXISTS `file`;
CREATE TABLE `file` (
                        `id` bigint(20) UNSIGNED NOT NULL COMMENT '主键ID',
                        `directory_id` bigint(20) UNSIGNED NOT NULL COMMENT '目录id',
                        `name` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT '文件名',
                        `type` varchar(64) NOT NULL COMMENT '文件类型',
                        `size` bigint(20) NOT NULL COMMENT '文件大小',
                        `sha` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT 'sha值',
                        `key` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT 'key值',
                        `src` varchar(256) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL COMMENT '文件路径',
                        `status` enum('PROGRESS','COMPLETED') DEFAULT 'PROGRESS' COMMENT '上传状态',
                        `upload_id` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL COMMENT '上传id',
                        `chunk_count` int(11) DEFAULT '1' COMMENT '切片数量',
                        `created_at` bigint(20) UNSIGNED DEFAULT NULL COMMENT '创建时间',
                        `updated_at` bigint(20) UNSIGNED DEFAULT NULL COMMENT '修改时间',
                        `deleted_at` bigint(20) UNSIGNED DEFAULT NULL COMMENT '删除时间'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

--
-- 转存表中的数据 `file`
--

INSERT INTO `file` (`id`, `directory_id`, `name`, `type`, `size`, `sha`, `key`, `src`, `status`, `upload_id`, `chunk_count`, `created_at`, `updated_at`, `deleted_at`) VALUES
                                                                                                                                                                           (8, 2, '1', 'png', 1, '385d37202ae8f08cd8ba429eb51b5422', '385d37202ae8f08cd8ba429eb51b5422.png', '2/385d37202ae8f08cd8ba429eb51b5422.png', 'COMPLETED', '15b37beb-b730-45fc-a238-1b8bc55bc677', 1, 1718105928, 1718106326, NULL),
                                                                                                                                                                           (9, 2, '2f18521c2d1cd5cc6fb8b939345f56d6d9b11fc89907b4070f96800b68455e58.png', 'png', 2, '1f27444925877922d71110d993edf590', '1f27444925877922d71110d993edf590.png', '2/1f27444925877922d71110d993edf590.png', 'COMPLETED', '5a4b7a92-ef82-46c2-ae54-0110f7cc1ce5', 1, 1718107618, 1718107618, NULL),
                                                                                                                                                                           (10, 2, '24acef3dbe2cc8eb776008fc133e4f73338a3644a6581763f35f3ffc71d22641.png', 'png', 4, '6dc607ee0b87559d8932377d46b9a3ea', '6dc607ee0b87559d8932377d46b9a3ea.png', '2/6dc607ee0b87559d8932377d46b9a3ea.png', 'COMPLETED', '19e7f55b-31ba-46b1-b22b-b105e64e5e7b', 1, 1718107618, 1718107618, NULL),
                                                                                                                                                                           (11, 2, '97252c871797d84d7f582df166ed07711834ec0c675a0d171df39770f3a93960.png', 'png', 1, 'd19faa31f4a04b52f802a465edf50d18', 'd19faa31f4a04b52f802a465edf50d18.png', '2/d19faa31f4a04b52f802a465edf50d18.png', 'COMPLETED', '15a49db9-8f29-4fd3-a834-5e45212e80ef', 1, 1718107618, 1718107618, NULL),
                                                                                                                                                                           (12, 2, 'apps.png', 'png', 3, '2a0786fe9127b8116bc30ed2ce9581e2', '2a0786fe9127b8116bc30ed2ce9581e2.png', '2/2a0786fe9127b8116bc30ed2ce9581e2.png', 'COMPLETED', '87aaa3f9-227e-4f99-8e60-a12ff216dbe4', 1, 1718107618, 1718107618, NULL),
                                                                                                                                                                           (13, 2, 'home-act.png', 'png', 4, '6d06733ef579fbcef68b9f95745a3e99', '6d06733ef579fbcef68b9f95745a3e99.png', '2/6d06733ef579fbcef68b9f95745a3e99.png', 'COMPLETED', '7f8a014b-45b2-4cd2-82a3-921b76390f2f', 1, 1718107618, 1718107618, NULL),
                                                                                                                                                                           (14, 2, 'user.png', 'png', 3, '36e2e87f7b73219343da52a28ba47eec', '36e2e87f7b73219343da52a28ba47eec.png', '2/36e2e87f7b73219343da52a28ba47eec.png', 'COMPLETED', 'fcfe8ebd-b17d-40e1-b1da-0453370c6cba', 1, 1718107618, 1718107618, NULL),
                                                                                                                                                                           (15, 7, '2a0786fe9127b8116bc30ed2ce9581e2.png', 'png', 3, '2a0786fe9127b8116bc30ed2ce9581e2', '2a0786fe9127b8116bc30ed2ce9581e2.png', '7/2a0786fe9127b8116bc30ed2ce9581e2.png', 'COMPLETED', '87aaa3f9-227e-4f99-8e60-a12ff216dbe4_copy_634a0634', 1, 1718391488, 1718391488, NULL),
                                                                                                                                                                           (23, 7, '36e2e87f7b73219343da52a28ba47eec.png', 'png', 3, '36e2e87f7b73219343da52a28ba47eec', '36e2e87f7b73219343da52a28ba47eec.png', '7/36e2e87f7b73219343da52a28ba47eec.png', 'COMPLETED', 'fcfe8ebd-b17d-40e1-b1da-0453370c6cba_copy_0b75d1f5', 1, 1718392578, 1718392578, NULL),
                                                                                                                                                                           (25, 10, '36e2e87f7b73219343da52a28ba47eec.png', 'png', 3, '36e2e87f7b73219343da52a28ba47eec', '36e2e87f7b73219343da52a28ba47eec.png', '10/36e2e87f7b73219343da52a28ba47eec.png', 'COMPLETED', 'fcfe8ebd-b17d-40e1-b1da-0453370c6cba_copy_d56102a6', 1, 1718438213, 1718438213, NULL),
                                                                                                                                                                           (26, 7, '2f18521c2d1cd5cc6fb8b939345f56d6d9b11fc89907b4070f96800b68455e58.png', 'png', 2, '1f27444925877922d71110d993edf590', '1f27444925877922d71110d993edf590.png', '7/1f27444925877922d71110d993edf590.png', 'COMPLETED', '5a4b7a92-ef82-46c2-ae54-0110f7cc1ce5_copy_ac48594c', 1, 1718702909, 1718702909, NULL),
                                                                                                                                                                           (27, 7, '5118bb9a26458eed86525a00b02c8bba8299dfa98244239858c50b0be431a069.png', 'png', 1, '385d37202ae8f08cd8ba429eb51b5422', '385d37202ae8f08cd8ba429eb51b5422.png', '7/385d37202ae8f08cd8ba429eb51b5422.png', 'COMPLETED', '15b37beb-b730-45fc-a238-1b8bc55bc677_copy_4d48306d', 1, 1718702921, 1718702921, NULL),
                                                                                                                                                                           (28, 7, '微信.png', 'png', 1, '2252554cf6309d2e53e95a5d40458d17', '2252554cf6309d2e53e95a5d40458d17.png', '7/2252554cf6309d2e53e95a5d40458d17.png', 'COMPLETED', '6f4c4024-9a0f-4938-a69f-b8501ac9a5e5', 1, 1718731014, 1718731014, NULL),
                                                                                                                                                                           (29, 7, 'QQ方形.png', 'png', 1, '49dc09f716382dd3f460daaba2649939', '49dc09f716382dd3f460daaba2649939.png', '7/49dc09f716382dd3f460daaba2649939.png', 'COMPLETED', '471ab6f8-1274-4c01-b111-e363aab1a5b7', 1, 1718731448, 1718731448, NULL),
                                                                                                                                                                           (30, 12, 'WeChatb091830b712a5267076580f6d295526a.jpg', 'jpg', 18, 'b091830b712a5267076580f6d295526a', 'b091830b712a5267076580f6d295526a.jpg', '12/b091830b712a5267076580f6d295526a.jpg', 'COMPLETED', '6394cc6f-1e69-46d9-8200-c45ad52d59b1', 1, 1719068664, 1719070492, NULL),
                                                                                                                                                                           (31, 2, 'WeChatb091830b712a5267076580f6d295526a.jpg', 'jpg', 18, 'b091830b712a5267076580f6d295526a', 'b091830b712a5267076580f6d295526a.jpg', '2/b091830b712a5267076580f6d295526a.jpg', 'COMPLETED', '6394cc6f-1e69-46d9-8200-c45ad52d59b1_copy_6c70fb41', 1, 1719070533, 1719070533, NULL),
                                                                                                                                                                           (32, 12, 'QQ方形.png', 'png', 1, '49dc09f716382dd3f460daaba2649939', '49dc09f716382dd3f460daaba2649939.png', '12/49dc09f716382dd3f460daaba2649939.png', 'COMPLETED', '471ab6f8-1274-4c01-b111-e363aab1a5b7_copy_5a701c49', 1, 1719070620, 1719070620, NULL),
                                                                                                                                                                           (33, 12, '微信.png', 'png', 1, '2463b25a2ca03d94825dc4b466716a0c', '2463b25a2ca03d94825dc4b466716a0c.png', '12/2463b25a2ca03d94825dc4b466716a0c.png', 'PROGRESS', 'ff81aa96-8dc3-4f0b-9077-6e409674b165', 1, 1719070655, 1719070655, NULL),
                                                                                                                                                                           (34, 12, '微信方.png', 'png', 2, 'b373234319cc81c55ddd81b8de001f11', 'b373234319cc81c55ddd81b8de001f11.png', '12/b373234319cc81c55ddd81b8de001f11.png', 'COMPLETED', '11ecafc3-6c8a-48f2-9e39-15decb13a0d6', 1, 1719070780, 1719070904, NULL);

-- --------------------------------------------------------

--
-- 表的结构 `gorm_init`
--
DROP TABLE IF EXISTS `gorm_init`;
CREATE TABLE `gorm_init` (
                             `id` int(10) UNSIGNED NOT NULL,
                             `init` tinyint(1) DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

--
-- 转存表中的数据 `gorm_init`
--

INSERT INTO `gorm_init` (`id`, `init`) VALUES
    (1, 1);

--
-- 转储表的索引
--

--
-- 表的索引 `chunk`
--
ALTER TABLE `chunk`
    ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `upload_id` (`upload_id`,`index`),
  ADD UNIQUE KEY `ui` (`upload_id`,`index`),
  ADD KEY `idx_chunk_created_at` (`created_at`);

--
-- 表的索引 `directory`
--
ALTER TABLE `directory`
    ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `pna` (`parent_id`,`name`),
  ADD KEY `idx_directory_created_at` (`created_at`),
  ADD KEY `idx_directory_updated_at` (`updated_at`);

--
-- 表的索引 `directory_closure`
--
ALTER TABLE `directory_closure`
    ADD PRIMARY KEY (`id`),
  ADD KEY `parent` (`parent`),
  ADD KEY `children` (`children`);

--
-- 表的索引 `export`
--
ALTER TABLE `export`
    ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `sha` (`sha`,`user_id`);

--
-- 表的索引 `file`
--
ALTER TABLE `file`
    ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `sha` (`sha`,`directory_id`),
  ADD UNIQUE KEY `upload_id` (`upload_id`),
  ADD KEY `deleted_at` (`deleted_at`),
  ADD KEY `directory_id` (`directory_id`);

--
-- 表的索引 `gorm_init`
--
ALTER TABLE `gorm_init`
    ADD PRIMARY KEY (`id`);

--
-- 在导出的表使用AUTO_INCREMENT
--

--
-- 使用表AUTO_INCREMENT `chunk`
--
ALTER TABLE `chunk`
    MODIFY `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '分片id', AUTO_INCREMENT=166;

--
-- 使用表AUTO_INCREMENT `directory`
--
ALTER TABLE `directory`
    MODIFY `id` bigint(20) UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID', AUTO_INCREMENT=13;

--
-- 使用表AUTO_INCREMENT `directory_closure`
--
ALTER TABLE `directory_closure`
    MODIFY `id` bigint(20) UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID', AUTO_INCREMENT=2;

--
-- 使用表AUTO_INCREMENT `export`
--
ALTER TABLE `export`
    MODIFY `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID', AUTO_INCREMENT=9;

--
-- 使用表AUTO_INCREMENT `file`
--
ALTER TABLE `file`
    MODIFY `id` bigint(20) UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID', AUTO_INCREMENT=35;

--
-- 使用表AUTO_INCREMENT `gorm_init`
--
ALTER TABLE `gorm_init`
    MODIFY `id` int(10) UNSIGNED NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=2;

--
-- 限制导出的表
--

--
-- 限制表 `directory_closure`
--
ALTER TABLE `directory_closure`
    ADD CONSTRAINT `directory_closure_ibfk_1` FOREIGN KEY (`children`) REFERENCES `directory` (`id`) ON DELETE CASCADE,
  ADD CONSTRAINT `directory_closure_ibfk_2` FOREIGN KEY (`parent`) REFERENCES `directory` (`id`) ON DELETE CASCADE;

--
-- 限制表 `file`
--
ALTER TABLE `file`
    ADD CONSTRAINT `file_ibfk_1` FOREIGN KEY (`directory_id`) REFERENCES `directory` (`id`);
COMMIT;

/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
