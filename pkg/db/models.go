package db

import "time"

// Domain represents a blocked domain entry
type Domain struct {
	DomainName string    `db:"domain_name"`
	CreatedAt  time.Time `db:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"`
}

// DomainIP represents an IP address associated with a blocked domain
type DomainIP struct {
	ID         int64     `db:"id"`
	DomainName string    `db:"domain_name"`
	IPAddress  string    `db:"ip_address"`
	CreatedAt  time.Time `db:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"`
}
