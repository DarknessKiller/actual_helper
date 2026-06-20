package main

import (
	"log"

	"actual-helper/internal/bootstrap"
	"actual-helper/internal/handlers"
	rytprov "actual-helper/internal/providers/ryt"
	tngprov "actual-helper/internal/providers/tng"
	"actual-helper/internal/services"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-fuego/fuego"
)

func main() {
	server := fuego.NewServer(
		fuego.WithEngineOptions(
			fuego.WithOpenAPIConfig(fuego.OpenAPIConfig{
				Info: &openapi3.Info{
					Title:       "Actual Helper",
					Description: "Converts bank/fintech transaction files (CSV or PDF) into Actual Budget-compatible CSV format.",
					Version:     "1.0.0",
				},
			}),
		),
	)

	registry, loader := bootstrap.Init(map[string]bootstrap.ProviderFactory{
		"tng": tngprov.New,
		"ryt": rytprov.New,
	})

	convertService := services.NewConvertService(registry, loader)
	handler := handlers.NewConvertHandler(convertService)
	handlers.RegisterConvertRoutes(server, handler)

	if err := server.Run(); err != nil {
		log.Fatal(err)
	}
}
