package pdfutil

type ExtractionMethod string

const (
	ExtractionMethodDigital   ExtractionMethod = "digital"
	ExtractionMethodPdftotext ExtractionMethod = "pdftotext"
	ExtractionMethodOCR       ExtractionMethod = "ocr"
)
