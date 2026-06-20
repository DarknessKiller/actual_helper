package main

import (
	"log"

	"actual-helper/internal/bootstrap"
	"actual-helper/internal/handlers"
	tngprov "actual-helper/internal/providers/tng"
	"actual-helper/internal/services"

	"github.com/go-fuego/fuego"
)

func main() {
	server := fuego.NewServer()

	registry, loader := bootstrap.Init(map[string]bootstrap.ProviderFactory{
		"tng": tngprov.New,
	})

	convertService := services.NewConvertService(registry, loader)
	handler := handlers.NewConvertHandler(convertService)
	handlers.RegisterConvertRoutes(server, handler)

	if err := server.Run(); err != nil {
		log.Fatal(err)
	}
}
