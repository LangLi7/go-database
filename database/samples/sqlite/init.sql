-- go-database Sample Database: SQLite
-- Schema: Task Management

CREATE TABLE IF NOT EXISTS projects (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    description TEXT,
    status TEXT DEFAULT 'active',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    priority TEXT DEFAULT 'medium',
    status TEXT DEFAULT 'todo',
    assignee TEXT,
    due_date DATE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    color TEXT DEFAULT '#3498db'
);

CREATE TABLE IF NOT EXISTS task_tags (
    task_id INTEGER NOT NULL,
    tag_id INTEGER NOT NULL,
    PRIMARY KEY (task_id, tag_id),
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
);

-- Sample Data
INSERT INTO projects (name, description, status) VALUES
    ('Website Redesign', 'Redesign the company website', 'active'),
    ('Mobile App', 'Develop iOS/Android app', 'active'),
    ('Database Migration', 'Migrate from MySQL to PostgreSQL', 'planning');

INSERT INTO tasks (project_id, title, description, priority, status, assignee, due_date) VALUES
    (1, 'Design mockups', 'Create Figma mockups for homepage', 'high', 'done', 'Alice', '2025-03-01'),
    (1, 'Implement header', 'Code the responsive header', 'medium', 'in_progress', 'Bob', '2025-03-10'),
    (1, 'Implement footer', 'Code the footer component', 'low', 'todo', 'Bob', '2025-03-15'),
    (2, 'Setup CI/CD', 'Configure GitHub Actions', 'high', 'done', 'Charlie', '2025-02-28'),
    (2, 'API integration', 'Connect to backend API', 'high', 'in_progress', 'Alice', '2025-03-20'),
    (3, 'Schema analysis', 'Analyze current MySQL schema', 'medium', 'done', 'Charlie', '2025-02-25'),
    (3, 'Migration script', 'Write migration scripts', 'high', 'todo', 'Alice', '2025-03-30');

INSERT INTO tags (name, color) VALUES
    ('urgent', '#e74c3c'),
    ('frontend', '#3498db'),
    ('backend', '#2ecc71'),
    ('bug', '#f39c12'),
    ('feature', '#9b59b6');

INSERT INTO task_tags (task_id, tag_id) VALUES
    (1, 2), (1, 5),
    (2, 2),
    (3, 2),
    (4, 3), (4, 1),
    (5, 3),
    (6, 3),
    (7, 3), (7, 1);
