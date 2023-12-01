CREATE TABLE `directory` (
    `id` bigint NOT NULL AUTO_INCREMENT COMMENT '目录id',
    `parent_id` bigint DEFAULT NULL COMMENT '父级目录',
    `name` varchar(128) NOT NULL COMMENT '文件名称',
    `created_at` bigint unsigned DEFAULT NULL COMMENT '创建时间',
    `updated_at` bigint unsigned DEFAULT NULL COMMENT '修改时间',
    `app` varchar(32) NOT NULL COMMENT '所属应用',
    PRIMARY KEY (`id`),
    unique index(`name`)
) ENGINE=InnoDB  DEFAULT CHARSET=utf8mb4;

CREATE TABLE `file` (
                        `id` bigint NOT NULL AUTO_INCREMENT COMMENT '文件id',
                        `directory_id` bigint NOT NULL COMMENT '目录id',
                        `name` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT '文件名',
                        `type` varchar(64) NOT NULL COMMENT '文件类型',
                        `size` bigint NOT NULL COMMENT '文件大小',
                        `sha` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT 'sha值',
                        `src` varchar(256) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL COMMENT '文件路径',
                        `storage` varchar(32) NOT NULL COMMENT '存储引擎',
                        `status` enum('PROGRESS','COMPLETED','FAILED') DEFAULT 'PROGRESS' COMMENT '上传状态',
                        `upload_id` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL COMMENT '上传id',
                        `chunk_count` int DEFAULT '1' COMMENT '切片数量',
                        `created_at` bigint unsigned DEFAULT NULL COMMENT '创建时间',
                        `updated_at` bigint unsigned DEFAULT NULL COMMENT '修改时间',
                        `deleted_at` bigint unsigned DEFAULT NULL COMMENT '删除时间',
                        PRIMARY KEY (`id`),
                        UNIQUE KEY `sha` (`sha`,`directory_id`),
                        UNIQUE KEY `name` (`name`,`directory_id`),
                        UNIQUE KEY `upload_id` (`upload_id`),
                        KEY `deleted_at` (`deleted_at`),
                        KEY `directory_id` (`directory_id`),
                        CONSTRAINT `file_ibfk_1` FOREIGN KEY (`directory_id`) REFERENCES `directory` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;


CREATE TABLE `chunk` (
                         `id` bigint NOT NULL AUTO_INCREMENT COMMENT '分片id',
                         `upload_id` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT '上传id',
                         `index` int NOT NULL COMMENT '分片下标',
                         `size` bigint NOT NULL COMMENT '分片大小',
                         `sha` varchar(128) NOT NULL COMMENT 'sha值',
                         `data` mediumblob NOT NULL COMMENT '分片数据',
                         `created_at` bigint unsigned DEFAULT NULL COMMENT '创建时间',
                         PRIMARY KEY (`id`),
                         UNIQUE KEY `upload_id` (`upload_id`,`index`),
                         CONSTRAINT `chunk_ibfk_1` FOREIGN KEY (`upload_id`) REFERENCES `file` (`upload_id`) ON DELETE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=61 DEFAULT CHARSET=utf8mb4;