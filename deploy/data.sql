/*
 Navicat Premium Data Transfer

 Source Server         : dev
 Source Server Type    : MySQL
 Source Server Version : 80200
 Source Host           : localhost:3306
 Source Schema         : resource

 Target Server Type    : MySQL
 Target Server Version : 80200
 File Encoding         : 65001

 Date: 03/05/2024 22:58:44
*/

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- ----------------------------
-- Table structure for chunk
-- ----------------------------
DROP TABLE IF EXISTS `chunk`;
CREATE TABLE `chunk` (
                         `id` bigint NOT NULL AUTO_INCREMENT COMMENT '分片id',
                         `upload_id` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT '上传id',
                         `index` bigint NOT NULL COMMENT '切片下标',
                         `size` bigint NOT NULL COMMENT '切片大小',
                         `sha` varchar(128) NOT NULL COMMENT '切片sha',
                         `data` mediumblob NOT NULL COMMENT '切片数据',
                         `created_at` bigint unsigned DEFAULT NULL COMMENT '创建时间',
                         PRIMARY KEY (`id`),
                         UNIQUE KEY `upload_id` (`upload_id`,`index`),
                         UNIQUE KEY `ui` (`upload_id`,`index`),
                         KEY `idx_chunk_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ----------------------------
-- Table structure for directory
-- ----------------------------
DROP TABLE IF EXISTS `directory`;
CREATE TABLE `directory` (
                             `id` int unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
                             `created_at` bigint DEFAULT NULL COMMENT '创建时间',
                             `updated_at` bigint DEFAULT NULL COMMENT '修改时间',
                             `parent_id` int unsigned NOT NULL COMMENT '父id',
                             `name` varchar(128) NOT NULL COMMENT '目录名称',
                             `app` varchar(32) NOT NULL COMMENT '所属应用',
                             PRIMARY KEY (`id`),
                             UNIQUE KEY `pna` (`parent_id`,`name`,`app`),
                             KEY `idx_directory_created_at` (`created_at`),
                             KEY `idx_directory_updated_at` (`updated_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='目录信息';

-- ----------------------------
-- Table structure for export
-- ----------------------------
DROP TABLE IF EXISTS `export`;
CREATE TABLE `export` (
                          `id` bigint NOT NULL AUTO_INCREMENT COMMENT '文件id',
                          `user_id` bigint NOT NULL COMMENT '用户id',
                          `name` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT '文件名',
                          `size` bigint NOT NULL DEFAULT '0' COMMENT '文件大小',
                          `version` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT '版本',
                          `src` varchar(256) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL COMMENT '文件路径',
                          `reason` varchar(512) NOT NULL DEFAULT '' COMMENT '错误原因',
                          `status` varchar(32) NOT NULL DEFAULT '' COMMENT '导出状态',
                          `created_at` bigint unsigned DEFAULT NULL COMMENT '创建时间',
                          `updated_at` bigint unsigned DEFAULT NULL COMMENT '修改时间',
                          PRIMARY KEY (`id`),
                          UNIQUE KEY `version` (`version`,`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='导出任务';

-- ----------------------------
-- Table structure for file
-- ----------------------------
DROP TABLE IF EXISTS `file`;
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
                        UNIQUE KEY `upload_id` (`upload_id`),
                        KEY `deleted_at` (`deleted_at`),
                        KEY `directory_id` (`directory_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='文件信息';

-- ----------------------------
-- Table structure for gorm_init
-- ----------------------------
DROP TABLE IF EXISTS `gorm_init`;
CREATE TABLE `gorm_init` (
                             `id` int unsigned NOT NULL AUTO_INCREMENT,
                             `init` tinyint(1) DEFAULT NULL,
                             PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=0 DEFAULT CHARSET=utf8mb4;

SET FOREIGN_KEY_CHECKS = 1;
