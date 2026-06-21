package config

import (
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-fuego/fuego"
)

func NewFuegoServer(env Env) *fuego.Server {
	isProd := env.Environment == "production"

	host := "localhost"
	if isProd {
		host = "0.0.0.0"
	}

	server := fuego.NewServer(
		fuego.WithEngineOptions(
			fuego.WithOpenAPIConfig(fuego.OpenAPIConfig{
				Info: &openapi3.Info{
					Title:       "Actual Helper",
					Description: "Converts bank/fintech transaction files (CSV or PDF) into Actual Budget-compatible CSV format.",
					Version:     Version,
				},
				DisableDefaultServer: isProd,
			}),
		),
		fuego.WithAddr(fmt.Sprintf("%s:%d", host, env.Port)),
	)

	if isProd {
		server.OpenAPI.Description().Servers = []*openapi3.Server{
			{
				URL:         env.PublicURL,
				Description: "Production server",
			},
		}
	}

	return server
}
