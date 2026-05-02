package provider

import (
	"fmt"
	"log"
	"sync"

	"noveltts/internal/model"
)

type Registry struct {
	mu        sync.RWMutex
	providers map[string]Provider
	configs   map[string]model.ProviderConfig
}

func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
		configs:   make(map[string]model.ProviderConfig),
	}
}

func (r *Registry) LoadFromConfig(cfg *model.AppConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.providers = make(map[string]Provider)
	r.configs = make(map[string]model.ProviderConfig)

	for _, pc := range cfg.Providers {
		r.configs[pc.Name] = pc
		prov, err := r.createProvider(pc)
		if err != nil {
			log.Printf("[registry] skip provider %s: %v", pc.Name, err)
			continue
		}
		r.providers[pc.Name] = prov
		log.Printf("[registry] registered provider: %s (%s)", pc.Name, pc.Type)
	}
	return nil
}

func (r *Registry) createProvider(cfg model.ProviderConfig) (Provider, error) {
	switch cfg.Type {
	case "openai_compatible":
		return NewOpenAIProvider(cfg), nil
	case "minimax":
		return NewMiniMaxProvider(cfg), nil
	case "mimo":
		return NewMiMoProvider(cfg), nil
	case "doubao":
		return NewDoubaoProvider(cfg), nil
	default:
		return nil, fmt.Errorf("unknown provider type: %s", cfg.Type)
	}
}

func (r *Registry) Get(name string) (Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[name]
	return p, ok
}

func (r *Registry) GetAll() map[string]Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[string]Provider, len(r.providers))
	for k, v := range r.providers {
		result[k] = v
	}
	return result
}

func (r *Registry) GetDefault(cfg *model.AppConfig) (Provider, error) {
	if cfg.Defaults.Provider != "" {
		if p, ok := r.Get(cfg.Defaults.Provider); ok {
			return p, nil
		}
		return nil, fmt.Errorf("default provider %q not found", cfg.Defaults.Provider)
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, p := range r.providers {
		return p, nil
	}
	return nil, fmt.Errorf("no providers configured")
}
