-- 创建数据库
CREATE DATABASE IF NOT EXISTS go_book_transaction DEFAULT CHARSET utf8mb4 COLLATE utf8mb4_general_ci;

USE go_book_transaction;

-- 账户表
CREATE TABLE IF NOT EXISTS accounts (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL UNIQUE,
    balance BIGINT NOT NULL DEFAULT 0 COMMENT '余额，单位为分',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 插入测试数据
INSERT INTO accounts (user_id, balance) VALUES 
    ('A', 100000),
    ('B', 50000),
    ('C', 200000);

-- 验证数据
SELECT * FROM accounts;
