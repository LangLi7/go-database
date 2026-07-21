-- E-Commerce Plattform — Beispieldaten

INSERT INTO categories (name, slug, description, sort_order) VALUES
('Elektronik', 'elektronik', 'Computer, Smartphones, Zubehör', 1),
('Kleidung', 'kleidung', 'Mode für Damen und Herren', 2),
('Bücher', 'buecher', 'Fachbücher, Romane, E-Books', 3),
('Haus & Garten', 'haus-garten', 'Möbel, Dekoration, Werkzeug', 4);

INSERT INTO products (sku, name, description, price, cost_price, stock_qty, category_id) VALUES
('ELE-001', 'Laptop Pro 15\"', 'Leistungsstarker Laptop mit 32GB RAM, 1TB SSD', 1299.00, 950.00, 15, 1),
('ELE-002', 'Smartphone X3', '6.7\" OLED Display, 256GB, 5G', 899.00, 650.00, 30, 1),
('ELE-003', 'USB-C Hub 7in1', 'Multiport-Adapter mit HDMI, USB-A, SD', 39.90, 18.00, 100, 1),
('KLE-001', 'Designer-Jeans Slim Fit', 'Hochwertige Denim-Jeans, schwarz', 89.90, 45.00, 50, 2),
('KLE-002', 'Wollpullover Classic', 'Kaschmir-Mix, marineblau', 119.00, 60.00, 35, 2),
('BUC-001', 'Clean Architecture', 'Robert C. Martin - Software-Architektur', 49.99, 25.00, 100, 3),
('BUC-002', 'Designing Data-Intensive Applications', 'Martin Kleppmann - Datenbanksysteme', 54.99, 28.00, 75, 3),
('GAR-001', 'Gartenstuhl Set 2er', 'Klappbare Alu-Gartenstühle, anthrazit', 159.00, 85.00, 25, 4);

INSERT INTO customers (email, first_name, last_name, phone, is_vip) VALUES
('max.mustermann@example.com', 'Max', 'Mustermann', '+491701234567', true),
('anna.schmidt@example.com', 'Anna', 'Schmidt', '+491712345678', false),
('tech.corp@firma.de', 'TechCorp', 'GmbH', '+4969123456', true),
('lisa.wolf@web.de', 'Lisa', 'Wolf', '+491731234567', false);

INSERT INTO addresses (customer_id, type, street, city, postal_code, country, is_default) VALUES
(1, 'both', 'Hauptstraße 42', 'Berlin', '10115', 'Deutschland', true),
(2, 'shipping', 'Mühlenweg 7', 'Hamburg', '20095', 'Deutschland', true),
(3, 'billing', 'Industriestraße 1', 'Frankfurt', '60327', 'Deutschland', true);

INSERT INTO orders (order_number, customer_id, status, total_net, total_gross, shipping_cost, payment_method) VALUES
('ORD-2026-0001', 1, 'delivered', 1299.00, 1546.81, 0, 'Kreditkarte'),
('ORD-2026-0002', 2, 'shipped', 988.90, 1176.79, 4.99, 'PayPal'),
('ORD-2026-0003', 1, 'pending', 49.99, 59.49, 2.99, 'Rechnung');

INSERT INTO order_items (order_id, product_id, quantity, unit_price, total_price) VALUES
(1, 1, 1, 1299.00, 1299.00),
(2, 2, 1, 899.00, 899.00),
(2, 5, 1, 89.90, 89.90),
(3, 6, 1, 49.99, 49.99);
