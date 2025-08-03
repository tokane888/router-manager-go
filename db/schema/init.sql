-- Initialize router_manager database schema
-- Create domains table to store blocked domain names
CREATE TABLE IF NOT EXISTS domains (
    domain_name VARCHAR(255) PRIMARY KEY,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create domain_ips table to store IP addresses resolved from domains
CREATE TABLE IF NOT EXISTS domain_ips (
    id BIGSERIAL PRIMARY KEY,
    domain_name VARCHAR(255) NOT NULL,
    ip_address VARCHAR(45) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_domain_ips_domain_name FOREIGN KEY (domain_name) REFERENCES domains(domain_name) ON DELETE CASCADE
);

-- Create index for better query performance
CREATE INDEX IF NOT EXISTS idx_domain_ips_domain_name ON domain_ips(domain_name);

CREATE INDEX IF NOT EXISTS idx_domain_ips_ip_address ON domain_ips(ip_address);

-- Create update trigger for updated_at columns
CREATE
OR REPLACE FUNCTION update_updated_at_column() RETURNS TRIGGER AS $ $ BEGIN NEW.updated_at = CURRENT_TIMESTAMP;

RETURN NEW;

END;

$ $ language 'plpgsql';

-- Apply triggers to automatically update updated_at columns
CREATE TRIGGER update_domains_updated_at BEFORE
UPDATE
    ON domains FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_domain_ips_updated_at BEFORE
UPDATE
    ON domain_ips FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
