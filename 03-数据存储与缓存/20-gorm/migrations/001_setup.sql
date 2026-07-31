-- 创建数据库
CREATE DATABASE IF NOT EXISTS go_book_gorm DEFAULT CHARSET utf8mb4 COLLATE utf8mb4_general_ci;

USE go_book_gorm;

-- 用户表
CREATE TABLE IF NOT EXISTS users (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) NOT NULL,
    email VARCHAR(100) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    phone VARCHAR(20) NULL,
    status TINYINT NOT NULL DEFAULT 1 COMMENT '0:禁用 1:启用',
    balance DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
    version INT NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at DATETIME NULL,
    active_username VARCHAR(50)
        GENERATED ALWAYS AS (IF(deleted_at IS NULL, username, NULL)) STORED,
    active_email VARCHAR(100)
        GENERATED ALWAYS AS (IF(deleted_at IS NULL, email, NULL)) STORED,

    UNIQUE INDEX uk_active_username (active_username),
    UNIQUE INDEX uk_active_email (active_email),
    INDEX idx_email (email),
    INDEX idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 订单表
CREATE TABLE IF NOT EXISTS orders (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    order_no VARCHAR(64) NOT NULL,
    user_id BIGINT NOT NULL,
    status TINYINT NOT NULL DEFAULT 0 COMMENT '0:待支付 1:已支付 2:已完成',
    total DECIMAL(10, 2) NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    UNIQUE INDEX idx_order_no (order_no),
    INDEX idx_user_id (user_id),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 订单明细表
CREATE TABLE IF NOT EXISTS order_items (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    order_id BIGINT NOT NULL,
    product_id BIGINT NOT NULL,
    quantity INT NOT NULL,
    price DECIMAL(10, 2) NOT NULL,
    
    INDEX idx_order_id (order_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 插入测试数据
INSERT INTO users (username, email, password_hash, balance) VALUES
    ('alice', 'alice@example.com', 'hashed_password', 1000.00),
    ('bob', 'bob@example.com', 'hashed_password', 500.00),
    ('charlie', 'charlie@example.com', 'hashed_password', 2000.00);

INSERT INTO orders (order_no, user_id, status, total) VALUES
    ('ORD001', 1, 1, 100.00),
    ('ORD002', 1, 0, 200.00),
    ('ORD003', 2, 2, 150.00);

INSERT INTO order_items (order_id, product_id, quantity, price) VALUES
    (1, 1, 2, 50.00),
    (2, 2, 1, 200.00),
    (3, 3, 3, 50.00);
