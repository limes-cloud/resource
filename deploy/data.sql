

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

CREATE TABLE `directory` (
  `id` bigint(20) NOT NULL COMMENT '目录id',
  `parent_id` int(10) UNSIGNED NOT NULL COMMENT '父id',
  `name` varchar(128) NOT NULL COMMENT '目录名称',
  `created_at` bigint(20) UNSIGNED DEFAULT NULL COMMENT '创建时间',
  `updated_at` bigint(20) UNSIGNED DEFAULT NULL COMMENT '修改时间',
  `app` varchar(32) NOT NULL COMMENT '所属应用'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

--
-- 转存表中的数据 `directory`
--

INSERT INTO `directory` (`id`, `parent_id`, `name`, `created_at`, `updated_at`, `app`) VALUES
(4, 3, 'logo', 1706545229, 1706545229, 'UserCenter'),
(9, 5, 'logo', 1706545606, 1706545606, 'UserCenter'),
(10, 0, 'resource', 1706601622, 1706601622, 'PartyAffairs'),
(11, 10, 'banner', 1706601623, 1706601623, 'PartyAffairs'),
(12, 0, 'news', 1706605371, 1706605371, 'PartyAffairs'),
(13, 12, 'cover', 1706605371, 1706605371, 'PartyAffairs'),
(14, 0, '123', 1706666064, 1706666064, '2'),
(15, 0, 'app', 1708946491, 1708946491, 'UserCenter'),
(16, 15, 'logo', 1708946491, 1708946491, 'UserCenter'),
(17, 0, 'channel', 1708946613, 1708946613, 'UserCenter'),
(18, 17, 'logo', 1708946613, 1708946613, 'UserCenter'),
(19, 0, '1', 1708965432, 1708965432, 'Resource'),
(20, 0, 'test', 1709538782, 1709538782, 'Manager');

-- --------------------------------------------------------

--
-- 表的结构 `file`
--

CREATE TABLE `file` (
  `id` bigint(20) NOT NULL COMMENT '文件id',
  `directory_id` bigint(20) NOT NULL COMMENT '目录id',
  `name` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT '文件名',
  `type` varchar(64) NOT NULL COMMENT '文件类型',
  `size` bigint(20) NOT NULL COMMENT '文件大小',
  `sha` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT 'sha值',
  `src` varchar(256) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL COMMENT '文件路径',
  `storage` varchar(32) NOT NULL COMMENT '存储引擎',
  `status` enum('PROGRESS','COMPLETED','FAILED') DEFAULT 'PROGRESS' COMMENT '上传状态',
  `upload_id` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL COMMENT '上传id',
  `chunk_count` int(11) DEFAULT '1' COMMENT '切片数量',
  `created_at` bigint(20) UNSIGNED DEFAULT NULL COMMENT '创建时间',
  `updated_at` bigint(20) UNSIGNED DEFAULT NULL COMMENT '修改时间',
  `deleted_at` bigint(20) UNSIGNED DEFAULT NULL COMMENT '删除时间'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

--
-- 转存表中的数据 `file`
--

INSERT INTO `file` (`id`, `directory_id`, `name`, `type`, `size`, `sha`, `src`, `storage`, `status`, `upload_id`, `chunk_count`, `created_at`, `updated_at`, `deleted_at`) VALUES
(6, 4, '密码.png', 'png', 768, '97252c871797d84d7f582df166ed07711834ec0c675a0d171df39770f3a93960', '97252c871797d84d7f582df166ed07711834ec0c675a0d171df39770f3a93960.png', 'local', 'COMPLETED', '8befa740-f395-443b-b274-1b4fe475823b', 1, 1706545229, 1706545230, NULL),
(7, 4, '短信.png', 'png', 729, '5118bb9a26458eed86525a00b02c8bba8299dfa98244239858c50b0be431a069', '5118bb9a26458eed86525a00b02c8bba8299dfa98244239858c50b0be431a069.png', 'local', 'COMPLETED', '28d4007b-8a2a-4756-9660-1f65d80f3d6c', 1, 1706545244, 1706545244, NULL),
(10, 11, '1666153317708_135.png', 'png', 91575, '1740d203a3b650f04e29ffe4619e4f1c5c3d77575ae6493754f164cfcff15fbc', '1740d203a3b650f04e29ffe4619e4f1c5c3d77575ae6493754f164cfcff15fbc.png', 'local', 'COMPLETED', '74107091-500a-4976-92a1-e1451e25f64f', 1, 1706601679, 1706601679, NULL),
(13, 11, '634a8a0b12778.png', 'png', 115279, 'b7ecd9860f1525466e236db39d7dcf4405ba476cf8ac31298274c3fce39d2c22', 'b7ecd9860f1525466e236db39d7dcf4405ba476cf8ac31298274c3fce39d2c22.png', 'local', 'COMPLETED', 'c5d1a3d0-097c-4244-8135-9a7c34a77060', 1, 1706606682, 1706606682, NULL),
(14, 11, 'level-good.png', 'png', 14380, '8f65687e87904b8cbbc705c99b3c10f64f65f24aa4599ac09c3cd2c0bbd474a1', '8f65687e87904b8cbbc705c99b3c10f64f65f24aa4599ac09c3cd2c0bbd474a1.png', 'local', 'COMPLETED', '1ea28d07-5567-43dc-951a-a04567049a7f', 1, 1706623497, 1706623498, NULL),
(16, 14, 'practice.png', 'png', 2625, '5dc254e5c4f4d58199519bee20625e4335dd797e12ff4fe55cdd736deeb2fcb1', '5dc254e5c4f4d58199519bee20625e4335dd797e12ff4fe55cdd736deeb2fcb1.png', 'local', 'COMPLETED', 'e61982ba-1412-4fd4-9512-02ea3813d02b', 1, 1706671265, 1706671265, NULL),
(19, 12, 'completed.png', 'png', 2563, '808bafcdb429a29e3c70d3901b0c8195c6328b838428a01965a729f3c39f69fc', '808bafcdb429a29e3c70d3901b0c8195c6328b838428a01965a729f3c39f69fc.png', 'local', 'COMPLETED', 'd1a7ebcc-4e60-4271-a487-a3e34094f887', 1, 1706795662, 1706795662, NULL),
(20, 16, '选中-特色党建.png', 'png', 3740, '24acef3dbe2cc8eb776008fc133e4f73338a3644a6581763f35f3ffc71d22641', '24acef3dbe2cc8eb776008fc133e4f73338a3644a6581763f35f3ffc71d22641.png', 'local', 'COMPLETED', '919d3602-d8f2-40d2-afcc-0d8b8693a3a4', 4, 1708946491, 1708946492, NULL),
(21, 18, '短信.png', 'png', 729, '5118bb9a26458eed86525a00b02c8bba8299dfa98244239858c50b0be431a069', '5118bb9a26458eed86525a00b02c8bba8299dfa98244239858c50b0be431a069.png', 'local', 'COMPLETED', NULL, 1, 1708946613, 1706545244, NULL),
(22, 18, '密码.png', 'png', 768, '97252c871797d84d7f582df166ed07711834ec0c675a0d171df39770f3a93960', '97252c871797d84d7f582df166ed07711834ec0c675a0d171df39770f3a93960.png', 'local', 'COMPLETED', NULL, 1, 1708946628, 1706545230, NULL),
(27, 19, '即将到期待办任务 (1).png', 'png', 985, 'bec4f346c4d9a1aad3bc4128974c14bf10db8f83126d8aaeb61177ec76fd4fa2', 'bec4f346c4d9a1aad3bc4128974c14bf10db8f83126d8aaeb61177ec76fd4fa2.png', 'local', 'COMPLETED', '141ae213-86b3-43c3-bbcd-55bc565adb7a', 1, 1708965441, 1708965441, NULL),
(28, 20, '1111', 'txt', 3, 'f6e0a1e2ac41945a9aa7ff8a8aaa0cebc12a3bcc981a929ad5cf810a090e11ae', 'f6e0a1e2ac41945a9aa7ff8a8aaa0cebc12a3bcc981a929ad5cf810a090e11ae.txt', 'local', 'COMPLETED', 'f5673f29-9833-41af-acee-31755d74cdd7', 1, 1709538841, 1709538907, NULL);

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
  ADD UNIQUE KEY `parent_id` (`parent_id`,`name`),
  ADD UNIQUE KEY `pna` (`parent_id`,`name`,`app`),
  ADD KEY `idx_directory_created_at` (`created_at`),
  ADD KEY `idx_directory_updated_at` (`updated_at`);

--
-- 表的索引 `file`
--
ALTER TABLE `file`
  ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `sha` (`sha`,`directory_id`),
  ADD UNIQUE KEY `name` (`name`,`directory_id`),
  ADD UNIQUE KEY `upload_id` (`upload_id`),
  ADD KEY `deleted_at` (`deleted_at`),
  ADD KEY `directory_id` (`directory_id`);

--
-- 在导出的表使用AUTO_INCREMENT
--

--
-- 使用表AUTO_INCREMENT `chunk`
--
ALTER TABLE `chunk`
  MODIFY `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '分片id', AUTO_INCREMENT=65;

--
-- 使用表AUTO_INCREMENT `directory`
--
ALTER TABLE `directory`
  MODIFY `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '目录id', AUTO_INCREMENT=21;

--
-- 使用表AUTO_INCREMENT `file`
--
ALTER TABLE `file`
  MODIFY `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '文件id', AUTO_INCREMENT=29;

--
-- 限制导出的表
--

--
-- 限制表 `chunk`
--
ALTER TABLE `chunk`
  ADD CONSTRAINT `chunk_ibfk_1` FOREIGN KEY (`upload_id`) REFERENCES `file` (`upload_id`) ON DELETE CASCADE;

--
-- 限制表 `file`
--
ALTER TABLE `file`
  ADD CONSTRAINT `file_ibfk_1` FOREIGN KEY (`directory_id`) REFERENCES `directory` (`id`);
COMMIT;

/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
