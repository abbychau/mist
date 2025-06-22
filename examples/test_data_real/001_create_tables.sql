-- Create companies table
CREATE TABLE companies (
    id INT AUTO_INCREMENT PRIMARY KEY,
    corporate_name VARCHAR(255) NOT NULL,
    representative VARCHAR(255) NOT NULL,
    phone_number VARCHAR(20) NOT NULL,
    postal_code VARCHAR(10) NOT NULL,
    address TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_companies_corporate_name (corporate_name)
);

-- Create users table
CREATE TABLE users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    company_id INT NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE CASCADE,
    INDEX idx_users_email (email),
    INDEX idx_users_company_id (company_id)
);

-- Create business_partners table
CREATE TABLE business_partners (
    id INT AUTO_INCREMENT PRIMARY KEY,
    company_id INT NOT NULL,
    corporate_name VARCHAR(255) NOT NULL,
    representative VARCHAR(255) NOT NULL,
    phone_number VARCHAR(20) NOT NULL,
    postal_code VARCHAR(10) NOT NULL,
    address TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE CASCADE,
    INDEX idx_business_partners_company_id (company_id),
    INDEX idx_business_partners_corporate_name (corporate_name)
);

-- Create business_partner_bank_accounts table
CREATE TABLE business_partner_bank_accounts (
    id INT AUTO_INCREMENT PRIMARY KEY,
    business_partner_id INT NOT NULL,
    bank_name VARCHAR(255) NOT NULL,
    branch_name VARCHAR(255) NOT NULL,
    account_number VARCHAR(20) NOT NULL,
    account_name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (business_partner_id) REFERENCES business_partners(id) ON DELETE CASCADE,
    INDEX idx_bank_accounts_business_partner_id (business_partner_id)
);

-- Create invoices table
CREATE TABLE invoices (
    id INT AUTO_INCREMENT PRIMARY KEY,
    company_id INT NOT NULL,
    business_partner_id INT NOT NULL,
    issue_date DATE NOT NULL,
    payment_amount DECIMAL(15, 2) NOT NULL,
    fee DECIMAL(15, 2) NOT NULL,
    fee_rate DECIMAL(5, 4) NOT NULL DEFAULT 0.0400,
    consumption_tax DECIMAL(15, 2) NOT NULL,
    consumption_tax_rate DECIMAL(5, 4) NOT NULL DEFAULT 0.1000,
    invoice_amount DECIMAL(15, 2) NOT NULL,
    payment_due_date DATE NOT NULL,
    status ENUM('unprocessed', 'processing', 'paid', 'error') NOT NULL DEFAULT 'unprocessed',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE CASCADE,
    FOREIGN KEY (business_partner_id) REFERENCES business_partners(id) ON DELETE CASCADE,
    INDEX idx_invoices_company_id (company_id),
    INDEX idx_invoices_business_partner_id (business_partner_id),
    INDEX idx_invoices_payment_due_date (payment_due_date),
    INDEX idx_invoices_status (status),
    INDEX idx_invoices_created_at (created_at)
);
