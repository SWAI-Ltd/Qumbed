package mesh

// Relay forwards messages without storing or reading payload (zero-knowledge).
// It only sees topic and recipient key ID for routing; no persistence or queue.
type Relay struct {
	addr string
}

// NewRelay creates a relay reference (connects on demand)
func NewRelay(addr string) *Relay {
	return &Relay{addr: addr}
}
