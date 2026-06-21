package main

import (
	"log"

	"actual_helper/internal/bootstrap"
	"actual_helper/internal/config"
	"actual_helper/internal/handlers"
	rytprov "actual_helper/internal/providers/ryt"
	tngprov "actual_helper/internal/providers/tng"
	"actual_helper/internal/ratelimit"
	"actual_helper/internal/services"

	"github.com/go-fuego/fuego"
)

func main() {
	registry, loader, env := bootstrap.Init(map[string]bootstrap.ProviderFactory{
		"tng": tngprov.New,
		"ryt": rytprov.New,
	})

	server := config.NewFuegoServer(env)

	convertService := services.NewConvertService(registry, loader)
	handler := handlers.NewConvertHandler(convertService)
	fuego.Use(server, ratelimit.Middleware)
	handlers.RegisterConvertRoutes(server, handler)
	handlers.RegisterFrontendRoutes(server.Mux, nil)

	if err := server.Run(); err != nil {
		log.Fatal(err)
	}
}
