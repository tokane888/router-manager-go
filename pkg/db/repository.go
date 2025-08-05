package db

import (
	"errors"
	"fmt"

	"github.com/lib/pq"
	"go.uber.org/zap"
)

// Domain repository operations

// CreateDomain inserts a new domain into the database
func (db *DB) CreateDomain(domainName string) error {
	query := `INSERT INTO domains (domain_name) VALUES ($1)`
	_, err := db.conn.Exec(query, domainName)
	if err != nil {
		// Check if it's a PostgreSQL unique constraint violation (error code 23505)
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			db.log.Warn("Domain already exists", zap.String("domain", domainName))
			return fmt.Errorf("failed to create domain %s: %w", domainName, ErrDomainAlreadyExists)
		}

		db.log.Error("Failed to create domain", zap.String("domain", domainName), zap.Error(err))
		return fmt.Errorf("failed to create domain %s: %w", domainName, err)
	}

	db.log.Info("Domain created successfully", zap.String("domain", domainName))
	return nil
}

// GetAllDomains retrieves all domains
func (db *DB) GetAllDomains() ([]Domain, error) {
	query := `SELECT domain_name, created_at, updated_at FROM domains ORDER BY domain_name`

	rows, err := db.conn.Query(query)
	if err != nil {
		db.log.Error("Failed to get all domains", zap.Error(err))
		return nil, fmt.Errorf("failed to get all domains: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			db.log.Error("Failed to close rows", zap.Error(err))
		}
	}()

	var domains []Domain
	for rows.Next() {
		var domain Domain
		err := rows.Scan(
			&domain.DomainName,
			&domain.CreatedAt,
			&domain.UpdatedAt,
		)
		if err != nil {
			db.log.Error("Failed to scan domain row", zap.Error(err))
			return nil, fmt.Errorf("failed to scan domain row: %w", err)
		}
		domains = append(domains, domain)
	}

	if err := rows.Err(); err != nil {
		db.log.Error("Failed to iterate domain rows", zap.Error(err))
		return nil, fmt.Errorf("failed to iterate domain rows: %w", err)
	}

	return domains, nil
}

// Domain IP repository operations

// CreateDomainIP inserts a new IP address for a domain
func (db *DB) CreateDomainIP(domainName, ipAddress string) error {
	query := `INSERT INTO domain_ips (domain_name, ip_address) VALUES ($1, $2)`
	_, err := db.conn.Exec(query, domainName, ipAddress)
	if err != nil {
		// postgresのユニークキー制約(error code 23505)に抵触していないか確認
		// 抵触している場合domain, ipペアが登録済み
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			db.log.Warn("Domain IP already exists",
				zap.String("domain", domainName),
				zap.String("ip", ipAddress))
			return fmt.Errorf("failed to create domain IP %s for %s: %w", ipAddress, domainName, ErrDomainIPAlreadyExists)
		}

		db.log.Error("Failed to create domain IP",
			zap.String("domain", domainName),
			zap.String("ip", ipAddress),
			zap.Error(err))
		return fmt.Errorf("failed to create domain IP %s for %s: %w", ipAddress, domainName, err)
	}

	db.log.Info("Domain IP created successfully",
		zap.String("domain", domainName),
		zap.String("ip", ipAddress))
	return nil
}

// GetDomainIPs retrieves all IP addresses for a domain
func (db *DB) GetDomainIPs(domainName string) ([]DomainIP, error) {
	query := `SELECT id, domain_name, ip_address, created_at, updated_at 
			  FROM domain_ips WHERE domain_name = $1 ORDER BY domain_name`

	rows, err := db.conn.Query(query, domainName)
	if err != nil {
		db.log.Error("Failed to get domain IPs", zap.String("domain", domainName), zap.Error(err))
		return nil, fmt.Errorf("failed to get domain IPs for %s: %w", domainName, err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			db.log.Error("Failed to close rows", zap.Error(err))
		}
	}()

	var domainIPs []DomainIP
	for rows.Next() {
		var domainIP DomainIP
		err := rows.Scan(
			&domainIP.ID,
			&domainIP.DomainName,
			&domainIP.IPAddress,
			&domainIP.CreatedAt,
			&domainIP.UpdatedAt,
		)
		if err != nil {
			db.log.Error("Failed to scan domain IP row", zap.Error(err))
			return nil, fmt.Errorf("failed to scan domain IP row: %w", err)
		}
		domainIPs = append(domainIPs, domainIP)
	}

	if err := rows.Err(); err != nil {
		db.log.Error("Failed to iterate domain IP rows", zap.Error(err))
		return nil, fmt.Errorf("failed to iterate domain IP rows: %w", err)
	}

	return domainIPs, nil
}

// DeleteDomainIP removes a specific IP address for a domain
func (db *DB) DeleteDomainIP(domainName, ipAddress string) error {
	query := `DELETE FROM domain_ips WHERE domain_name = $1 AND ip_address = $2`
	result, err := db.conn.Exec(query, domainName, ipAddress)
	if err != nil {
		db.log.Error("Failed to delete domain IP",
			zap.String("domain", domainName),
			zap.String("ip", ipAddress),
			zap.Error(err))
		return fmt.Errorf("failed to delete domain IP %s for %s: %w", ipAddress, domainName, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		db.log.Error("Failed to get rows affected",
			zap.String("domain", domainName),
			zap.String("ip", ipAddress),
			zap.Error(err))
		return fmt.Errorf("failed to get rows affected for domain IP %s: %w", ipAddress, err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("domain IP %s for %s not found", ipAddress, domainName)
	}

	db.log.Info("Domain IP deleted successfully",
		zap.String("domain", domainName),
		zap.String("ip", ipAddress))
	return nil
}

// GetAllDomainIPs retrieves all domain IP entries
func (db *DB) GetAllDomainIPs() ([]DomainIP, error) {
	query := `SELECT id, domain_name, ip_address, created_at, updated_at 
			  FROM domain_ips ORDER BY domain_name, created_at DESC`

	rows, err := db.conn.Query(query)
	if err != nil {
		db.log.Error("Failed to get all domain IPs", zap.Error(err))
		return nil, fmt.Errorf("failed to get all domain IPs: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			db.log.Error("Failed to close rows", zap.Error(err))
		}
	}()

	var domainIPs []DomainIP
	for rows.Next() {
		var domainIP DomainIP
		err := rows.Scan(
			&domainIP.ID,
			&domainIP.DomainName,
			&domainIP.IPAddress,
			&domainIP.CreatedAt,
			&domainIP.UpdatedAt,
		)
		if err != nil {
			db.log.Error("Failed to scan domain IP row", zap.Error(err))
			return nil, fmt.Errorf("failed to scan domain IP row: %w", err)
		}
		domainIPs = append(domainIPs, domainIP)
	}

	if err := rows.Err(); err != nil {
		db.log.Error("Failed to iterate domain IP rows", zap.Error(err))
		return nil, fmt.Errorf("failed to iterate domain IP rows: %w", err)
	}

	return domainIPs, nil
}
