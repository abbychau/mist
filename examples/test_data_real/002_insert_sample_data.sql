-- Insert sample data for testing

-- Insert sample companies
INSERT INTO companies (corporate_name, representative, phone_number, postal_code, address) VALUES
('Tech Solutions Inc.', 'John Smith', '03-1234-5678', '100-0001', 'Tokyo, Chiyoda-ku, Chiyoda 1-1-1'),
('Digital Services Corp.', 'Jane Doe', '03-8765-4321', '150-0002', 'Tokyo, Shibuya-ku, Shibuya 2-2-2');

-- Insert sample users (passwords are hashed for 'password123')
INSERT INTO users (company_id, full_name, email, password) VALUES
(1, 'Alice Johnson', 'alice@techsolutions.com', '$2a$10$rR6tqOZEOjgCEWXNDXz8uOhXqGKOQGUfxWVJYJ8eQqPKKFjqFQEXS'),
(2, 'Bob Wilson', 'bob@digitalservices.com', '$2a$10$rR6tqOZEOjgCEWXNDXz8uOhXqGKOQGUfxWVJYJ8eQqPKKFjqFQEXS');

-- Insert sample business partners
INSERT INTO business_partners (company_id, corporate_name, representative, phone_number, postal_code, address) VALUES
(1, 'Supplier A Ltd.', 'Michael Brown', '03-1111-2222', '101-0001', 'Tokyo, Chiyoda-ku, Marunouchi 1-1-1'),
(1, 'Vendor B Corp.', 'Sarah Davis', '03-3333-4444', '102-0002', 'Tokyo, Chiyoda-ku, Nihonbashi 2-2-2'),
(2, 'Partner C Inc.', 'David Wilson', '03-5555-6666', '103-0003', 'Tokyo, Chuo-ku, Ginza 3-3-3');

-- Insert sample bank accounts
INSERT INTO business_partner_bank_accounts (business_partner_id, bank_name, branch_name, account_number, account_name) VALUES
(1, 'Tokyo Bank', 'Shibuya Branch', '1234567890', 'Supplier A Ltd.'),
(2, 'Mizuho Bank', 'Shinjuku Branch', '0987654321', 'Vendor B Corp.'),
(3, 'MUFG Bank', 'Ginza Branch', '1122334455', 'Partner C Inc.');

-- Insert sample invoices
INSERT INTO invoices (company_id, business_partner_id, issue_date, payment_amount, fee, fee_rate, consumption_tax, consumption_tax_rate, invoice_amount, payment_due_date, status) VALUES
(1, 1, '2024-01-15', 100000.00, 4000.00, 0.0400, 400.00, 0.1000, 104400.00, '2024-02-15', 'unprocessed'),
(1, 2, '2024-01-20', 50000.00, 2000.00, 0.0400, 200.00, 0.1000, 52200.00, '2024-02-20', 'processing'),
(2, 3, '2024-01-25', 75000.00, 3000.00, 0.0400, 300.00, 0.1000, 78300.00, '2024-02-25', 'paid');
