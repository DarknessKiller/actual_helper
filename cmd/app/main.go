package main

import (
	"log"

	"actual-helper/internal/bootstrap"
	"actual-helper/internal/handlers"
	"actual-helper/internal/services"

	"github.com/go-fuego/fuego"
)

func main() {
	server := fuego.NewServer()

	registry, loader := bootstrap.Init()

	convertService := services.NewConvertService(registry, loader)
	handler := handlers.NewConvertHandler(convertService)
	handlers.RegisterConvertRoutes(server, handler)

	if err := server.Run(); err != nil {
		log.Fatal(err)
	}
}
