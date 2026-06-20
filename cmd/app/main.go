package main

import (
	"log"

	"actual-helper/internal/handlers"
	"actual-helper/internal/providers"
	tngprov "actual-helper/internal/providers/tng"
	"actual-helper/internal/services"

	"github.com/go-fuego/fuego"
)

func main() {
	server := fuego.NewServer()

	registry := providers.NewRegistry()
	registry.Register(tngprov.New())

	convertService := services.NewConvertService(registry)
	handler := handlers.NewConvertHandler(convertService)
	handlers.RegisterConvertRoutes(server, handler)

	if err := server.Run(); err != nil {
		log.Fatal(err)
	}
}
