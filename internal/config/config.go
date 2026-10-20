package config

import (
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"sync"
	"time"

	"actual_helper/internal/models"
)

type ProviderConfig struct {
	ExcludeKeywords []string              `json:"exclude_keywords"`
	IncludeKeywords []string              `json:"include_keywords"`
	Categories      []models.CategoryRule `json:"categories"`
	AccountMappings map[string]string     `json:"account_mappings"`
}

type config struct {
	Global    ProviderConfig            `json:"global"`
	Providers map[string]ProviderConfig `json:"providers"`
}

// Loader holds the active provider config in memory. By default nothing is
// loaded: ProviderConfig returns empty tuning until ApplyConfig is called.
// The file at ProviderConfigPath is read only by the GET /config download
// endpoint (see SampleConfig); it is never auto-applied to providers.
//
// Per-session tuning lives in the sessions map: the HTTP handler stores rules
// under the caller's cookie id and expires them lazily (no goroutine poller).
type Loader struct {
	mu     sync.Mutex
	loaded bool
	config config

	sessions    map[string]sessionEntry
	sessionTTL  time.Duration
	maxSessions int
}

type sessionEntry struct {
	rules   config
	expires time.Time
}

func NewLoader() *Loader {
	return &Loader{
		sessions:    map[string]sessionEntry{},
		sessionTTL:  24 * time.Hour,
		maxSessions: 1024,
	}
}

// ApplyConfig parses a user-supplied config JSON document and stores it in
// memory. Subsequent ProviderConfig calls return the merged values. Invalid
// JSON leaves the existing config untouched and returns the parse error.
func (l *Loader) ApplyConfig(data []byte) error {
	var cfg config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
	}

	l.mu.Lock()
	l.config = cfg
	l.loaded = true
	l.mu.Unlock()

	slog.Info("provider config applied")
	return nil
}

// ClearConfig drops the in-memory config so providers run with empty tuning.
func (l *Loader) ClearConfig() {
	l.mu.Lock()
	l.config = config{}
	l.loaded = false
	l.mu.Unlock()

	slog.Info("provider config cleared")
}

func (l *Loader) ProviderConfig(name string) ProviderConfig {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.loaded {
		return ProviderConfig{}
	}

	return l.config.providerConfig(name)
}

// ApplySessionConfig parses a user-supplied config JSON document and stores it
// for one session (cookie id). Session rules are merged over the global config
// on lookup; invalid JSON leaves the session's existing rules untouched.
func (l *Loader) ApplySessionConfig(sid string, data []byte) error {
	var cfg config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	l.pruneSessionsLocked(now)
	if len(l.sessions) >= l.maxSessions {
		return ErrTooManySessions
	}
	l.sessions[sid] = sessionEntry{rules: cfg, expires: now.Add(l.sessionTTL)}
	return nil
}

// ErrTooManySessions is returned by ApplySessionConfig when the session store
// is at capacity (bounded memory: a public deployment cannot be grown without
// bound by strangers posting configs).
var ErrTooManySessions = errors.New("too many active config sessions")

// SessionProviderConfig returns the tuning for one provider for one session:
// global rules first, then the session rules on top. Empty when the session is
// unknown or expired; expired sessions are dropped lazily on lookup.
func (l *Loader) SessionProviderConfig(sid string, name string) ProviderConfig {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry, ok := l.sessions[sid]
	if ok && time.Now().After(entry.expires) {
		delete(l.sessions, sid)
		ok = false
	}
	if !ok {
		if !l.loaded {
			return ProviderConfig{}
		}
		return l.config.providerConfig(name)
	}

	merged := l.config.providerConfig(name)
	session := entry.rules.providerConfig(name)

	merged.ExcludeKeywords = append(merged.ExcludeKeywords, session.ExcludeKeywords...)
	merged.IncludeKeywords = append(merged.IncludeKeywords, session.IncludeKeywords...)
	merged.Categories = append(merged.Categories, session.Categories...)

	if merged.AccountMappings == nil {
		merged.AccountMappings = session.AccountMappings
	} else {
		for account, target := range session.AccountMappings {
			merged.AccountMappings[account] = target
		}
	}

	return merged
}

// ClearSession drops one session's stored rules.
func (l *Loader) ClearSession(sid string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.sessions, sid)
}

func (l *Loader) pruneSessionsLocked(now time.Time) {
	for sid, entry := range l.sessions {
		if now.After(entry.expires) {
			delete(l.sessions, sid)
		}
	}
}

// SampleConfig reads the sample config file at path. It is used by the
// GET /config download endpoint and never mutates Loader state. Returns an
// error when path is empty or the file cannot be read.
func SampleConfig(path string) ([]byte, error) {
	if path == "" {
		return nil, os.ErrNotExist
	}
	return os.ReadFile(path)
}

func (cfg config) providerConfig(name string) ProviderConfig {
	var pc ProviderConfig
	pc.ExcludeKeywords = append(pc.ExcludeKeywords, cfg.Global.ExcludeKeywords...)
	pc.IncludeKeywords = append(pc.IncludeKeywords, cfg.Global.IncludeKeywords...)
	pc.Categories = append(pc.Categories, cfg.Global.Categories...)

	if p, ok := cfg.Providers[name]; ok {
		pc.ExcludeKeywords = append(pc.ExcludeKeywords, p.ExcludeKeywords...)
		pc.IncludeKeywords = append(pc.IncludeKeywords, p.IncludeKeywords...)
		pc.Categories = append(pc.Categories, p.Categories...)
		pc.AccountMappings = p.AccountMappings
	}

	return pc
}

// SetSessionTTL overrides the session expiry (used by tests).
func (l *Loader) SetSessionTTL(ttl time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.sessionTTL = ttl
}

// SessionLimit overrides the session store capacity (used by tests).
func (l *Loader) SessionLimit(max uint8) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.maxSessions = int(max)
}
