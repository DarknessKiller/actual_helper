package main

import (
	"actual_helper/frontend"
	"actual_helper/internal/config"
	"actual_helper/internal/handlers"
	"io/fs"
	"log"
)

func main() {
	env := config.LoadEnv()
	server := config.NewFuegoServer(env)
	dist, err := fs.Sub(frontend.FS, "dist")
	if err != nil {
		dist = nil
	}
	handlers.RegisterFrontendRoutes(server.Mux, dist)
	if err := server.Run(); err != nil {
		log.Fatal(err)
	}
}
