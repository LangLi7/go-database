-- go-database Sample Database: MariaDB
-- Schema: Inventory Management

CREATE TABLE IF NOT EXISTS warehouses (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    location VARCHAR(255),
    capacity INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS suppliers (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    contact_name VARCHAR(255),
    contact_email VARCHAR(255),
    phone VARCHAR(50),
    rating DECIMAL(2,1) DEFAULT 0.0
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS inventory_items (
    id INT AUTO_INCREMENT PRIMARY KEY,
    warehouse_id INT NOT NULL,
    supplier_id INT,
    sku VARCHAR(100) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    quantity INT DEFAULT 0,
    min_stock INT DEFAULT 10,
    unit_price DECIMAL(10,2),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (warehouse_id) REFERENCES warehouses(id),
    FOREIGN KEY (supplier_id) REFERENCES suppliers(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS stock_movements (
    id INT AUTO_INCREMENT PRIMARY KEY,
    item_id INT NOT NULL,
    type ENUM('in', 'out', 'adjustment') NOT NULL,
    quantity INT NOT NULL,
    reason VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (item_id) REFERENCES inventory_items(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Sample Data
INSERT INTO warehouses (name, location, capacity) VALUES
    ('Main Warehouse', 'Berlin, Germany', 10000),
    ('East Warehouse', 'Warsaw, Poland', 5000),
    ('West Warehouse', 'Amsterdam, Netherlands', 7500);

INSERT INTO suppliers (name, contact_name, contact_email, phone, rating) VALUES
    ('TechSupply GmbH', 'Hans Schmidt', 'hans@techsupply.de', '+49-30-123456', 4.5),
    ('Global Parts Inc.', 'John Smith', 'john@globalparts.com', '+1-555-987654', 4.2);

INSERT INTO inventory_items (warehouse_id, supplier_id, sku, name, description, quantity, min_stock, unit_price) VALUES
    (1, 1, 'LAP-001', 'Laptop Standard', 'Business laptop 15\"', 25, 10, 899.99),
    (1, 1, 'MON-002', 'Monitor 27\"', '4K IPS Monitor', 15, 5, 449.99),
    (2, 2, 'KEY-003', 'Keyboard Wireless', 'Bluetooth keyboard', 50, 20, 79.99),
    (3, 2, 'MOU-004', 'Mouse Pad XL', 'Gaming mouse pad', 100, 30, 29.99);

INSERT INTO stock_movements (item_id, type, quantity, reason) VALUES
    (1, 'in', 50, 'Initial stock'),
    (1, 'out', 25, 'Order Fulfillment'),
    (2, 'in', 30, 'Restock'),
    (3, 'in', 100, 'New arrival'),
    (4, 'adjustment', -5, 'Damaged items removed');
