package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	dir := flag.String("dir", "./dist", "Static files directory")
	port := flag.String("port", "8080", "Port to listen on")
	flag.Parse()

	if _, err := os.Stat(*dir); os.IsNotExist(err) {
		log.Fatalf("Directory does not exist: %s", *dir)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Clean path
		path := filepath.Clean(r.URL.Path)
		if path == "/" {
			path = "/index.html"
		}

		// Try to serve the file
		fullPath := filepath.Join(*dir, path)
		if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
			// Set cache headers for static assets
			ext := strings.ToLower(filepath.Ext(fullPath))
			switch ext {
			case ".js", ".css", ".png", ".jpg", ".jpeg", ".gif", ".ico", ".svg", ".woff", ".woff2", ".mjs":
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			case ".html":
				w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
			}
			http.ServeFile(w, r, fullPath)
			return
		}

		// SPA fallback: serve index.html for any non-file path
		http.ServeFile(w, r, filepath.Join(*dir, "index.html"))
	})

	addr := ":" + *port
	fmt.Printf("Serving %s on http://0.0.0.0%s\n", *dir, addr)
	log.Fatal(http.ListenAndServe(addr, handler))
}
