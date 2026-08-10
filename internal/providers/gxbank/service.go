package gxbank

import (
	"actual_helper/internal/models"
	"actual_helper/internal/pdfutil"
	"actual_helper/internal/providers"
	"actual_helper/internal/providers/cardutil"
	"actual_helper/internal/rule"
	"context"
	"errors"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"
)

type GXBankProvider struct {
	engine         *rule.Engine
	mu             sync.RWMutex
	accountMapping map[string]string
}

func New(a, b []string, c []models.CategoryRule, d map[string]string) providers.Provider {
	return &GXBankProvider{engine: rule.NewEngine(a, b, c), accountMapping: d}
}
func (p *GXBankProvider) Reload(a, b []string, c []models.CategoryRule, d map[string]string) {
	p.engine.Reload(a, b, c)
	p.mu.Lock()
	p.accountMapping = d
	p.mu.Unlock()
}
func (p *GXBankProvider) Name() string { return "gxbank" }
func (p *GXBankProvider) ParseCSV(context.Context, io.Reader) ([]models.ActualBudgetReport, error) {
	return nil, errors.New("gxbank CSV parsing not supported; provider only supports PDF")
}
func (p *GXBankProvider) ParsePDFText(ctx context.Context, text string) ([]models.ActualBudgetReport, error) {
	a := ExtractAccountName(text)
	r, e := ParsePDFBlocks(text)
	if e != nil {
		return nil, e
	}
	out := p.toActualReports(ctx, slog.With("provider", "gxbank", "format", "pdf"), r, a)
	if len(out) == 0 {
		return nil, errors.New("no transactions found after filtering")
	}
	return out, nil
}
func (p *GXBankProvider) toActualReports(_ context.Context, _ *slog.Logger, rs []GXReport, a string) []models.ActualBudgetReport {
	p.mu.RLock()
	if m, ok := p.accountMapping[a]; ok {
		a = m
	}
	p.mu.RUnlock()
	var out []models.ActualBudgetReport
	for _, r := range rs {
		if p.shouldSkip(r.Description) {
			continue
		}
		d, e := parseDate(r.Date)
		if e != nil {
			continue
		}
		desc := strings.TrimSpace(cardutil.WhitespaceRe.ReplaceAllString(r.Description, " "))
		n, e := strconv.ParseFloat(strings.ReplaceAll(strings.TrimPrefix(strings.TrimPrefix(r.Amount, "+"), "-"), ",", ""), 64)
		if e != nil || n == 0 {
			continue
		}
		if !r.IsCredit {
			n = -n
		}
		g, c := p.matchCategory(desc)
		out = append(out, models.ActualBudgetReport{Account: a, Date: d.Format("2006-01-02"), Notes: desc, CategoryGroup: g, Category: c, Amount: strconv.FormatFloat(n, 'f', 2, 64)})
	}
	return out
}
func (p *GXBankProvider) shouldSkip(s string) bool                { return p.engine.ShouldSkip(s) }
func (p *GXBankProvider) matchCategory(s string) (string, string) { return p.engine.MatchCategory(s) }
func (p *GXBankProvider) ExtractionMethod() pdfutil.ExtractionMethod {
	return pdfutil.ExtractionMethodDigital
}
func parseDate(s string) (time.Time, error) {
	for _, f := range []string{"2 January 2006", "2 Jan 2006"} {
		if t, e := time.Parse(f, s); e == nil {
			return t, nil
		}
	}
	return time.Time{}, errors.New("invalid date format")
}
