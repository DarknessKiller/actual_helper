package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// writeJSON encodes v as the response body. The status and headers are already
// committed, so a write failure can only be logged, never returned.
func writeJSON(w http.ResponseWriter, v any) {
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Warn("encode response failed", "error", err)
	}
}
