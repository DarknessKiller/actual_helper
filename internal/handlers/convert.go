package handlers

import (
	"fmt"
	"log/slog"

	"actual-helper/internal/services"

	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/option"
)

type ConvertHandler struct {
	convertService *services.ConvertService
}

func NewConvertHandler(convertService *services.ConvertService) *ConvertHandler {
	return &ConvertHandler{convertService: convertService}
}

func (handler *ConvertHandler) Convert(c fuego.ContextNoBody) (any, error) {
	providerName := c.PathParam("provider")

	if err := c.Request().ParseMultipartForm(10 << 20); err != nil {
		return nil, fuego.BadRequestError{Title: "Invalid form", Detail: err.Error()}
	}

	file, header, err := c.Request().FormFile("file")
	if err != nil {
		return nil, fuego.BadRequestError{Title: "File required", Detail: err.Error()}
	}
	defer file.Close()

	filename := header.Filename
	contentType := header.Header.Get("Content-Type")

	slog.Info("request received", "provider", providerName, "filename", filename, "size", header.Size)

	csvBytes, err := handler.convertService.ConvertFile(c.Context(), providerName, file, filename, contentType, c.Request().FormValue("password"))
	if err != nil {
		return nil, fuego.InternalServerError{Title: "Conversion failed", Detail: err.Error()}
	}

	c.Response().Header().Set("Content-Type", "text/csv")
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s_actual_budget.csv", providerName))
	c.Response().Write(csvBytes)

	slog.Info("response sent", "provider", providerName, "bytes", len(csvBytes))
	return nil, nil
}

func RegisterConvertRoutes(server *fuego.Server, convertHandler *ConvertHandler) {
	fuego.Post(server, "/convert/{provider}", convertHandler.Convert,
		option.Summary("Convert provider transaction file to Actual Budget CSV"),
		option.Description("Upload a CSV or encrypted PDF transaction file from a supported provider and get back an Actual Budget compatible CSV."),
		option.Tags("convert"),
	)
}
