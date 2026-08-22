package config

import (
	"fmt"
	"time"

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

	// Fuego's default http.Server timeouts (30s) are too short for OCR, which
	// can run minutes on multi-page statements — the connection dies at 30s
	// while the handler keeps running, surfacing as a 503 to the client.
	// Bump to exceed the OCR budget (timeoutOCR = 5 min).
	server.Server.ReadTimeout = 10 * time.Minute
	server.Server.WriteTimeout = 10 * time.Minute
	server.Server.IdleTimeout = 5 * time.Minute

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
