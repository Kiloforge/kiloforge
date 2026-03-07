// Package port defines the interfaces (ports) that the service layer depends on.
//
// Adapters implement these interfaces and are injected into services at construction time.
//
// # Not-found sentinel error convention
//
// Port methods that look up a single entity by identifier return a domain sentinel error
// (e.g. domain.ErrProjectNotFound) when the entity does not exist. Callers can check
// with errors.Is(err, domain.ErrProjectNotFound).
package port
