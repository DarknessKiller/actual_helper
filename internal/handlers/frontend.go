package handlers

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

	"actual_helper/internal/config"
)

// RegisterFrontendRoutes registers an SPA-friendly static file handler on the given mux.
// distFS can be nil in development to serve from disk at frontend/dist/.
func RegisterFrontendRoutes(mux *http.ServeMux, distFS fs.FS) {
	devDir := "frontend/dist"

	if distFS == nil {
		if _, err := os.Stat(devDir); os.IsNotExist(err) {
			return
		}
	}

	mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"version": config.Version})
	})

	mux.Handle("/", spaHandler(distFS, devDir))
}

func spaHandler(distFS fs.FS, devDir string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// For non-root paths, check if the file exists first
		if r.URL.Path != "/" {
			clean := r.URL.Path[1:]
			exists := fileOnDisk(distFS, devDir, clean)
			if !exists {
				r.URL.Path = "/"
			}
		}

		if distFS != nil {
			http.FileServer(http.FS(distFS)).ServeHTTP(w, r)
		} else {
			http.FileServer(http.Dir(devDir)).ServeHTTP(w, r)
		}
	})
}

func fileOnDisk(distFS fs.FS, devDir, path string) bool {
	if distFS != nil {
		_, err := fs.Stat(distFS, path)
		return err == nil
	}
	_, err := os.Stat(filepath.Join(devDir, path))
	return err == nil
}
