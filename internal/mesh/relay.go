package mesh

// Relay is a broker that routes messages without reading payload (zero-knowledge).
// It only sees topic and recipient key ID for routing.
type Relay struct {
	addr string
}

// NewRelay creates a relay reference (connects on demand)
func NewRelay(addr string) *Relay {
	return &Relay{addr: addr}
}
