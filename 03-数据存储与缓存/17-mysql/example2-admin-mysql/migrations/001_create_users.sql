CREATE DATABASE IF NOT EXISTS admin_db
    DEFAULT CHARACTER SET utf8mb4
    DEFAULT COLLATE utf8mb4_0900_ai_ci;

USE admin_db;

-- 创建用户表
CREATE TABLE IF NOT EXISTS users (
    id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '用户ID',
    username VARCHAR(50) NOT NULL COMMENT '用户名',
    email VARCHAR(100) NOT NULL COMMENT '邮箱',
    password_hash VARCHAR(255) NOT NULL COMMENT '密码哈希（bcrypt）',
    phone VARCHAR(20) NULL COMMENT '手机号，保留国家码扩展空间',
    status TINYINT NOT NULL DEFAULT 1 COMMENT '状态：0-禁用 1-启用',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    deleted_at DATETIME NULL COMMENT '删除时间（软删除）',
    active_username VARCHAR(50) GENERATED ALWAYS AS (IF(deleted_at IS NULL, username, NULL)) STORED COMMENT '未删除用户名唯一键',
    active_email VARCHAR(100) GENERATED ALWAYS AS (IF(deleted_at IS NULL, email, NULL)) STORED COMMENT '未删除邮箱唯一键',
    
    -- 只约束未删除数据唯一，软删除后可以重新注册同一用户名或邮箱
    UNIQUE INDEX uk_users_active_username (active_username),
    UNIQUE INDEX uk_users_active_email (active_email),
    
    -- 后台列表查询：按状态筛选并按创建时间倒序
    INDEX idx_users_status_deleted_created_id (status, deleted_at, created_at DESC, id DESC),
    INDEX idx_users_deleted_created_id (deleted_at, created_at DESC, id DESC),
    INDEX idx_users_phone (phone)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户表';

-- 插入测试数据
INSERT IGNORE INTO users (username, email, password_hash, phone, status) VALUES
('admin', 'admin@example.com', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', '13800000000', 1),
('alice', 'alice@example.com', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', '13800138000', 1),
('bob', 'bob@example.com', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', '13900139000', 1);
