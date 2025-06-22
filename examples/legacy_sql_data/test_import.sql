-- Simple test SQL file for import functionality
CREATE TABLE test_users (
    id INT PRIMARY KEY AUTO_INCREMENT,
    username VARCHAR(50) NOT NULL,
    email VARCHAR(100),
    active BOOL
);

-- Insert some test data
INSERT INTO test_users (username, email, active) VALUES ('john_doe', 'john@example.com', true);
INSERT INTO test_users (username, email, active) VALUES ('jane_smith', 'jane@example.com', true);
INSERT INTO test_users (username, email, active) VALUES ('bob_wilson', 'bob@example.com', false);

-- Create an index
CREATE INDEX idx_username ON test_users (username);
