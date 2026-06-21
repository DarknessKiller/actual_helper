package main

import (
	"log"

	"actual-helper/internal/bootstrap"
	"actual-helper/internal/config"
	"actual-helper/internal/handlers"
	rytprov "actual-helper/internal/providers/ryt"
	tngprov "actual-helper/internal/providers/tng"
	"actual-helper/internal/ratelimit"
	"actual-helper/internal/services"

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

	if err := server.Run(); err != nil {
		log.Fatal(err)
	}
}
