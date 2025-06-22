-- Sample SQL file for demonstrating ImportSQLFile functionality
-- This file contains table creation and data insertion statements

-- Create departments table
CREATE TABLE departments (
    id INT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    budget FLOAT,
    location VARCHAR(50)
);

-- Create employees table
CREATE TABLE employees (
    id INT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(150),
    department_id INT,
    salary FLOAT,
    hire_date VARCHAR(20)
);

-- Create projects table
CREATE TABLE projects (
    id INT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    department_id INT,
    budget FLOAT,
    status VARCHAR(20)
);

-- Insert department data
INSERT INTO departments VALUES (1, 'Engineering', 500000.0, 'Building A');
INSERT INTO departments VALUES (2, 'Marketing', 200000.0, 'Building B');
INSERT INTO departments VALUES (3, 'Sales', 300000.0, 'Building C');
INSERT INTO departments VALUES (4, 'HR', 150000.0, 'Building A');

-- Insert employee data
INSERT INTO employees (name, email, department_id, salary, hire_date) VALUES 
    ('Alice Johnson', 'alice@company.com', 1, 85000.0, '2023-01-15'),
    ('Bob Smith', 'bob@company.com', 1, 92000.0, '2022-11-20'),
    ('Carol Davis', 'carol@company.com', 2, 65000.0, '2023-03-10'),
    ('David Wilson', 'david@company.com', 2, 70000.0, '2022-08-05'),
    ('Eve Brown', 'eve@company.com', 3, 75000.0, '2023-02-28'),
    ('Frank Miller', 'frank@company.com', 3, 78000.0, '2022-12-12'),
    ('Grace Lee', 'grace@company.com', 4, 60000.0, '2023-04-01');

-- Insert project data
INSERT INTO projects VALUES (1, 'Website Redesign', 1, 100000.0, 'Active');
INSERT INTO projects VALUES (2, 'Mobile App', 1, 150000.0, 'Planning');
INSERT INTO projects VALUES (3, 'Marketing Campaign', 2, 50000.0, 'Active');
INSERT INTO projects VALUES (4, 'Sales Training', 3, 25000.0, 'Completed');
INSERT INTO projects VALUES (5, 'HR System Upgrade', 4, 75000.0, 'Active');

-- Create some indexes for better performance
CREATE INDEX idx_employee_dept ON employees (department_id);
CREATE INDEX idx_project_dept ON projects (department_id);
CREATE INDEX idx_employee_salary ON employees (salary);
