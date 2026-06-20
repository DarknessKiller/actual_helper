package main

import (
	"fmt"
	"log"

	"actual-helper/internal/bootstrap"
	"actual-helper/internal/handlers"
	rytprov "actual-helper/internal/providers/ryt"
	tngprov "actual-helper/internal/providers/tng"
	"actual-helper/internal/ratelimit"
	"actual-helper/internal/services"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-fuego/fuego"
)

func main() {

	registry, loader, env := bootstrap.Init(map[string]bootstrap.ProviderFactory{
		"tng": tngprov.New,
		"ryt": rytprov.New,
	})

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
		fuego.WithAddr(fmt.Sprintf("0.0.0.0:%d", env.Port)),
	)

	if env.Environment == "production" {
		server.OpenAPI.Description().Servers = []*openapi3.Server{
			{
				URL:         env.PublicURL,
				Description: "Production server",
			},
		}
	}

	convertService := services.NewConvertService(registry, loader)
	handler := handlers.NewConvertHandler(convertService)
	fuego.Use(server, ratelimit.Middleware)
	handlers.RegisterConvertRoutes(server, handler)

	if err := server.Run(); err != nil {
		log.Fatal(err)
	}
}
