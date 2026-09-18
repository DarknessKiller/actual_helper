package handlers

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"actual_helper/internal/services"

	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/option"
	"github.com/go-fuego/fuego/param"
)

type ConvertRequestBody struct {
	File     []byte `json:"file" description:"Transaction file (CSV or encrypted PDF)"`
	Password string `json:"password,omitempty" description:"PDF decryption password (optional)"`
}

type ConvertResponseBody struct{}

type ConvertHandler struct {
	convertService *services.ConvertService
}

func NewConvertHandler(convertService *services.ConvertService) *ConvertHandler {
	return &ConvertHandler{convertService: convertService}
}

// maxUploadBytes caps the in-memory upload buffer. Uploads are never written to disk.
const maxUploadBytes int64 = 32 << 20

const maxPasswordBytes = 4 << 10

var (
	errFileRequired = errors.New("file required")
	errFileTooLarge = errors.New("file too large for in-memory processing")
)

type uploadedFile struct {
	data        []byte
	contentType string
	password    string
}

// parseUpload reads a multipart request into memory, capped at limit bytes.
// Unlike Request.FormFile it never spools parts to temp files.
func parseUpload(r *http.Request, limit int64) (*uploadedFile, error) {
	reader, err := r.MultipartReader()
	if err != nil {
		return nil, err
	}

	upload := &uploadedFile{}
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}

		switch part.FormName() {
		case "file":
			upload.contentType = part.Header.Get("Content-Type")
			data, err := io.ReadAll(io.LimitReader(part, limit+1))
			part.Close()
			if err != nil {
				return nil, err
			}
			if int64(len(data)) > limit {
				return nil, errFileTooLarge
			}
			zero(upload.data) // discard any earlier file part
			upload.data = data
		case "password":
			password, err := io.ReadAll(io.LimitReader(part, maxPasswordBytes))
			part.Close()
			if err != nil {
				return nil, err
			}
			upload.password = string(password)
		default:
			part.Close()
		}
	}

	if upload.data == nil {
		return nil, errFileRequired
	}

	return upload, nil
}

func (handler *ConvertHandler) Convert(c fuego.ContextWithBody[ConvertRequestBody]) (any, error) {
	providerName := c.PathParam("provider")

	upload, err := parseUpload(c.Request(), maxUploadBytes)
	if err != nil {
		if errors.Is(err, errFileTooLarge) {
			return nil, fuego.HTTPError{
				Status: http.StatusRequestEntityTooLarge,
				Title:  "File too large",
				Detail: fmt.Sprintf("file exceeds the %d MiB in-memory limit", maxUploadBytes>>20),
			}
		}
		return nil, fuego.BadRequestError{Title: "File required", Detail: err.Error()}
	}
	defer zero(upload.data)

	slog.Info("request received", "provider", providerName, "size", len(upload.data))

	csvBytes, err := handler.convertService.ConvertFile(c.Context(), providerName, upload.data, upload.contentType, upload.password)
	if err != nil {
		return nil, fuego.InternalServerError{Title: "Conversion failed", Detail: err.Error()}
	}
	defer zero(csvBytes)

	currentTime := time.Now()
	fileName := fmt.Sprintf("%s_actual_budget_%s.csv", providerName, currentTime.Local().Format("2006-01-02_150405"))

	c.Response().Header().Set("Content-Type", "text/csv")
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileName))
	c.Response().Write(csvBytes)

	slog.Info("response sent", "provider", providerName, "bytes", len(csvBytes))
	return nil, nil
}

// zero clears b so statement data does not linger in freed heap pages.
func zero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

func RegisterConvertRoutes(server *fuego.Server, convertHandler *ConvertHandler) {
	fuego.Post(server, "/convert/{provider}", convertHandler.Convert,
		option.Summary("Convert provider transaction file to Actual Budget CSV"),
		option.Description("Upload a CSV or encrypted PDF transaction file from a supported provider and get back an Actual Budget compatible CSV."),
		option.Tags("convert"),
		option.RequestContentType("multipart/form-data"),
		option.Path("provider", "any supported providers", param.Example("Touch n Go", "tng"), param.Example("RYT Bank", "ryt"), param.Example("HSBC Credit", "hsbccredit"), param.Example("HLB", "hlb"), param.Example("UOB Credit", "uobcredit"), param.Example("GX Bank", "gxbank")),
		option.AddResponse(200, "Successful conversion — returns a CSV file ready for Actual Budget import", fuego.Response{
			ContentTypes: []string{"text/csv"},
			Type:         ConvertResponseBody{},
		}),
	)
}
