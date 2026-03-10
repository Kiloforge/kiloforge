package domain

// SwarmCapacity represents the current agent swarm capacity.
type SwarmCapacity struct {
	Max       int `json:"max"`
	Active    int `json:"active"`
	Available int `json:"available"`
}
