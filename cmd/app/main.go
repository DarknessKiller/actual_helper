package main

import (
	"io/fs"
	"log"

	"actual_helper/frontend"
	"actual_helper/internal/bootstrap"
	"actual_helper/internal/config"
	"actual_helper/internal/handlers"
	hlbcreditprov "actual_helper/internal/providers/hlbcredit"
	hsbccreditprov "actual_helper/internal/providers/hsbccredit"
	rytprov "actual_helper/internal/providers/ryt"
	tngprov "actual_helper/internal/providers/tng"
	"actual_helper/internal/ratelimit"
	"actual_helper/internal/services"

	"github.com/go-fuego/fuego"
)

func main() {
	registry, loader, env := bootstrap.Init(map[string]bootstrap.ProviderFactory{
		"tng":        tngprov.New,
		"ryt":        rytprov.New,
		"hsbccredit": hsbccreditprov.New,
		"hlbcredit":  hlbcreditprov.New,
	})

	server := config.NewFuegoServer(env)

	convertService := services.NewConvertService(registry, loader)
	handler := handlers.NewConvertHandler(convertService)
	fuego.Use(server, ratelimit.Middleware)
	handlers.RegisterConvertRoutes(server, handler)

	dist, err := fs.Sub(frontend.FS, "dist")
	if err != nil {
		dist = nil
	}
	handlers.RegisterFrontendRoutes(server.Mux, dist)

	if err := server.Run(); err != nil {
		log.Fatal(err)
	}
}
