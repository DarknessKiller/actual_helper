package handlers

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"actual_helper/internal/config"
	"actual_helper/internal/providers"
)

// ConfigHandler exposes the provider-config lifecycle:
//
//	GET    /config  -> download the sample config file at env.ProviderConfigPath
//	POST   /config  -> upload user config JSON, apply tuning to all providers
//	DELETE /config  -> unload: clear tuning back to empty on all providers
//
// The config is held in memory on the config.Loader; the sample file is read
// on demand for downloads and never auto-applied.
type ConfigHandler struct {
	loader   *config.Loader
	registry *providers.Registry
	env      config.Env
}

func NewConfigHandler(loader *config.Loader, registry *providers.Registry, env config.Env) *ConfigHandler {
	return &ConfigHandler{loader: loader, registry: registry, env: env}
}

func (handler *ConfigHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handler.download(w, r)
	case http.MethodPost:
		handler.upload(w, r)
	case http.MethodDelete:
		handler.unload(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (handler *ConfigHandler) download(w http.ResponseWriter, _ *http.Request) {
	data, err := config.SampleConfig(handler.env.ProviderConfigPath)
	if err != nil {
		slog.Warn("sample config not available", "path", handler.env.ProviderConfigPath, "error", err)
		http.Error(w, "sample config not available", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", `attachment; filename="provider_config.example.json"`)
	w.Write(data)
}

func (handler *ConfigHandler) upload(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "invalid config", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if err := handler.loader.ApplyConfig(body); err != nil {
		slog.Warn("config upload rejected", "error", err)
		http.Error(w, "invalid config", http.StatusBadRequest)
		return
	}

	applied := handler.applyToProviders()
	slog.Info("config applied", "providers", applied)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"applied": applied})
}

func (handler *ConfigHandler) unload(w http.ResponseWriter, _ *http.Request) {
	handler.loader.ClearConfig()

	for _, provider := range handler.registry.All() {
		if cp, ok := provider.(providers.ConfigurableProvider); ok {
			cp.Reload(nil, nil, nil, nil)
		}
	}

	slog.Info("config cleared")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"cleared": true})
}

// applyToProviders pushes the merged per-provider tuning to every registered
// ConfigurableProvider and returns the names that received an update.
func (handler *ConfigHandler) applyToProviders() []string {
	applied := make([]string, 0)
	for _, provider := range handler.registry.All() {
		cp, ok := provider.(providers.ConfigurableProvider)
		if !ok {
			continue
		}
		pc := handler.loader.ProviderConfig(provider.Name())
		cp.Reload(pc.ExcludeKeywords, pc.IncludeKeywords, pc.Categories, pc.AccountMappings)
		applied = append(applied, provider.Name())
	}
	return applied
}

// RegisterConfigRoutes wires the config lifecycle endpoints on the given mux.
func RegisterConfigRoutes(mux *http.ServeMux, handler *ConfigHandler) {
	mux.HandleFunc("/config", handler.ServeHTTP)
}
