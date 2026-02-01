package oauth

import (
	"sync"
)

// registry implements ProviderRegistry with thread-safe provider management.
type registry struct {
	mu        sync.RWMutex
	providers map[string]AuthProvider
}

// NewRegistry creates a new provider registry.
func NewRegistry() ProviderRegistry {
	return &registry{
		providers: make(map[string]AuthProvider),
	}
}

// Register adds a provider to the registry.
func (r *registry) Register(provider AuthProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[provider.Name()] = provider
}

// Get retrieves a provider by name.
func (r *registry) Get(name string) (AuthProvider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[name]
	return p, ok
}

// List returns all registered provider names.
func (r *registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}
