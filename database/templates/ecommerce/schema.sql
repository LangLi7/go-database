-- E-Commerce Plattform — Schema
-- Kompatibel mit: PostgreSQL, MySQL, MariaDB, SQLite

CREATE TABLE IF NOT EXISTS categories (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        VARCHAR(100) NOT NULL,
    slug        VARCHAR(120) UNIQUE NOT NULL,
    description TEXT,
    parent_id   INTEGER REFERENCES categories(id),
    sort_order  INTEGER DEFAULT 0,
    is_active   BOOLEAN DEFAULT true,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS products (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    sku         VARCHAR(50) UNIQUE NOT NULL,
    name        VARCHAR(200) NOT NULL,
    description TEXT,
    price       DECIMAL(10,2) NOT NULL CHECK (price >= 0),
    cost_price  DECIMAL(10,2),
    stock_qty   INTEGER DEFAULT 0 CHECK (stock_qty >= 0),
    category_id INTEGER REFERENCES categories(id),
    image_url   VARCHAR(500),
    is_active   BOOLEAN DEFAULT true,
    weight_kg   DECIMAL(8,3),
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS customers (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    email       VARCHAR(255) UNIQUE NOT NULL,
    first_name  VARCHAR(100) NOT NULL,
    last_name   VARCHAR(100) NOT NULL,
    phone       VARCHAR(30),
    birth_date  DATE,
    is_vip      BOOLEAN DEFAULT false,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS addresses (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    customer_id INTEGER NOT NULL REFERENCES customers(id),
    type        VARCHAR(10) CHECK (type IN ('shipping','billing','both')),
    street      VARCHAR(200) NOT NULL,
    city        VARCHAR(100) NOT NULL,
    postal_code VARCHAR(20),
    country     VARCHAR(100) NOT NULL DEFAULT 'Deutschland',
    is_default  BOOLEAN DEFAULT false
);

CREATE TABLE IF NOT EXISTS orders (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    order_number    VARCHAR(30) UNIQUE NOT NULL,
    customer_id     INTEGER NOT NULL REFERENCES customers(id),
    status          VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending','confirmed','shipped','delivered','cancelled')),
    total_net       DECIMAL(12,2) NOT NULL,
    total_gross     DECIMAL(12,2) NOT NULL,
    currency        VARCHAR(3) DEFAULT 'EUR',
    shipping_cost   DECIMAL(10,2) DEFAULT 0,
    payment_method  VARCHAR(50),
    notes           TEXT,
    ordered_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    shipped_at      TIMESTAMP,
    delivered_at    TIMESTAMP
);

CREATE TABLE IF NOT EXISTS order_items (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    order_id    INTEGER NOT NULL REFERENCES orders(id),
    product_id  INTEGER NOT NULL REFERENCES products(id),
    quantity    INTEGER NOT NULL CHECK (quantity > 0),
    unit_price  DECIMAL(10,2) NOT NULL,
    total_price DECIMAL(12,2) NOT NULL
);

CREATE TABLE IF NOT EXISTS payments (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    order_id        INTEGER NOT NULL REFERENCES orders(id),
    amount          DECIMAL(12,2) NOT NULL,
    method          VARCHAR(50),
    transaction_id  VARCHAR(100),
    status          VARCHAR(20) DEFAULT 'pending',
    paid_at         TIMESTAMP
);

CREATE INDEX idx_products_category ON products(category_id);
CREATE INDEX idx_orders_customer ON orders(customer_id);
CREATE INDEX idx_order_items_order ON order_items(order_id);
CREATE INDEX idx_addresses_customer ON addresses(customer_id);
