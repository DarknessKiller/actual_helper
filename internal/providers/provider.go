package providers

import (
	"context"
	"io"

	"actual_helper/internal/models"
)

type Provider interface {
	Name() string
	ParseCSV(ctx context.Context, r io.Reader) ([]models.ActualBudgetReport, error)
	ParsePDFText(ctx context.Context, text string) ([]models.ActualBudgetReport, error)
}
