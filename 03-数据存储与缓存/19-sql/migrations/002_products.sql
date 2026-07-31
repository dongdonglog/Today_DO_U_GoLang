-- 创建数据库
CREATE DATABASE IF NOT EXISTS go_book_inventory DEFAULT CHARSET utf8mb4 COLLATE utf8mb4_general_ci;

USE go_book_inventory;

-- 商品表（带版本号，用于乐观锁）
CREATE TABLE IF NOT EXISTS products (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    stock INT NOT NULL DEFAULT 0,
    version INT NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 插入测试数据
INSERT INTO products (name, stock, version) VALUES 
    ('iPhone 15', 10),
    ('MacBook Pro', 5),
    ('AirPods', 100);

-- 验证数据
SELECT * FROM products;
