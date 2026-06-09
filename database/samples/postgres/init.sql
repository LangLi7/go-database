-- go-database Sample Database: PostgreSQL
-- Schema: E-Commerce Beispiel

CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    role VARCHAR(50) DEFAULT 'customer',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    price DECIMAL(10, 2) NOT NULL,
    stock INT DEFAULT 0,
    category VARCHAR(100),
    image_url TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS orders (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id),
    status VARCHAR(50) DEFAULT 'pending',
    total DECIMAL(10, 2) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS order_items (
    id SERIAL PRIMARY KEY,
    order_id INT NOT NULL REFERENCES orders(id),
    product_id INT NOT NULL REFERENCES products(id),
    quantity INT NOT NULL,
    price DECIMAL(10, 2) NOT NULL
);

-- Sample Data
INSERT INTO users (name, email, role) VALUES
    ('Alice Muster', 'alice@example.com', 'admin'),
    ('Bob Beispiel', 'bob@example.com', 'customer'),
    ('Charlie Test', 'charlie@example.com', 'customer');

INSERT INTO products (name, description, price, stock, category) VALUES
    ('Laptop Pro', 'High-end laptop', 1499.99, 10, 'Electronics'),
    ('Wireless Mouse', 'Ergonomic mouse', 49.99, 50, 'Accessories'),
    ('USB-C Hub', '7-port USB hub', 34.99, 30, 'Accessories'),
    ('Webcam HD', '1080p webcam', 89.99, 20, 'Electronics'),
    ('Mechanical Keyboard', 'RGB mechanical keyboard', 129.99, 15, 'Accessories');

INSERT INTO orders (user_id, status, total) VALUES
    (1, 'completed', 1549.98),
    (2, 'pending', 49.99),
    (3, 'shipped', 129.99);

INSERT INTO order_items (order_id, product_id, quantity, price) VALUES
    (1, 1, 1, 1499.99),
    (1, 2, 1, 49.99),
    (2, 2, 1, 49.99),
    (3, 5, 1, 129.99);
