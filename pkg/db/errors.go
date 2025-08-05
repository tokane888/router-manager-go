package db

import "errors"

// Domain-related errors
var (
	// ErrDomainAlreadyExists is returned when attempting to create a domain that already exists
	ErrDomainAlreadyExists = errors.New("domain already exists")

	// ErrDomainIPAlreadyExists is returned when attempting to create a domain IP that already exists
	ErrDomainIPAlreadyExists = errors.New("domain IP already exists")
)
