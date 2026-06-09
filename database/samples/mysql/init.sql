-- go-database Sample Database: MySQL
-- Schema: Blog-Plattform

CREATE TABLE IF NOT EXISTS authors (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    bio TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS categories (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    slug VARCHAR(100) NOT NULL UNIQUE,
    description TEXT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS posts (
    id INT AUTO_INCREMENT PRIMARY KEY,
    author_id INT NOT NULL,
    category_id INT,
    title VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL UNIQUE,
    content TEXT NOT NULL,
    published BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (author_id) REFERENCES authors(id),
    FOREIGN KEY (category_id) REFERENCES categories(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS comments (
    id INT AUTO_INCREMENT PRIMARY KEY,
    post_id INT NOT NULL,
    author_name VARCHAR(100) NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Sample Data
INSERT INTO authors (name, email, bio) VALUES
    ('Dora Explorer', 'dora@blog.com', 'Tech writer and explorer'),
    ('Max Developer', 'max@blog.com', 'Full-stack developer');

INSERT INTO categories (name, slug, description) VALUES
    ('Technology', 'technology', 'Tech articles and tutorials'),
    ('Design', 'design', 'UI/UX and design patterns'),
    ('Database', 'database', 'Database tips and tricks');

INSERT INTO posts (author_id, category_id, title, slug, content, published) VALUES
    (1, 1, 'Getting Started with Go', 'getting-started-go', 'Go is a statically typed language...', TRUE),
    (1, 3, 'PostgreSQL vs MySQL', 'postgres-vs-mysql', 'A comparison of two popular databases...', TRUE),
    (2, 2, 'Modern UI Design Patterns', 'modern-ui-patterns', 'Exploring modern design patterns...', FALSE);

INSERT INTO comments (post_id, author_name, content) VALUES
    (1, 'User123', 'Great article! Very helpful.'),
    (1, 'GoFan', 'I love Go, thanks for this.'),
    (2, 'DBAdmin', 'Very informative comparison.');
