package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"actual_helper/internal/config"
)

// ConfigHandler exposes the session-scoped provider-config lifecycle:
//
//	GET    /config  -> download the sample config file at env.ProviderConfigPath
//	POST   /config  -> store the posted rules for the caller's session (ah_sid cookie)
//	DELETE /config  -> clear the caller's own session rules
//
// Rules only ever apply to the holder of the session cookie, so no
// authentication is needed: a visitor can tune their own conversions but
// cannot affect anyone else's. The rules are held in memory per session id on
// the config.Loader and expire after the loader's session TTL.
type ConfigHandler struct {
	loader *config.Loader
	env    config.Env
}

func NewConfigHandler(loader *config.Loader, env config.Env) *ConfigHandler {
	return &ConfigHandler{loader: loader, env: env}
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

// upload stores the posted rules for the caller's session only.
func (handler *ConfigHandler) upload(w http.ResponseWriter, r *http.Request) {
	sid, err := ensureSession(w, r)
	if err != nil {
		slog.Error("session issue failed", "error", err)
		http.Error(w, "session error", http.StatusInternalServerError)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "invalid config", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if err := handler.loader.ApplySessionConfig(sid, body); err != nil {
		if errors.Is(err, config.ErrTooManySessions) {
			slog.Warn("config session store full", "error", err)
			http.Error(w, "too many active sessions; retry later", http.StatusTooManyRequests)
			return
		}
		slog.Warn("config upload rejected", "error", err)
		http.Error(w, "invalid config", http.StatusBadRequest)
		return
	}

	slog.Info("session config applied")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"stored": true})
}

// unload clears the caller's own session rules; a no-op without a cookie.
func (handler *ConfigHandler) unload(w http.ResponseWriter, r *http.Request) {
	sid := sessionID(r)
	if sid != "" {
		handler.loader.ClearSession(sid)
	}
	slog.Info("session config cleared")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"cleared": true})
}

// RegisterConfigRoutes wires the config lifecycle endpoints on the given mux.
func RegisterConfigRoutes(mux *http.ServeMux, handler *ConfigHandler) {
	mux.HandleFunc("/config", handler.ServeHTTP)
}
