package entity

// Domain represents a blocked domain with its resolved IPs
type Domain struct {
	Name string
	IPs  []string
}

// BlockingRule represents an nftables rule
type BlockingRule struct {
	IP     string
	Action string // "add" or "remove"
}
