package document

import (
	"bytes"
	"fmt"

	"github.com/ledongthuc/pdf"
)

type TextExtractor struct{}

func NewTextExtractor() *TextExtractor {
	return &TextExtractor{}
}

func (te *TextExtractor) ExtractFromPDF(data []byte) (string, error) {
	reader, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("failed to create PDF reader: %w", err)
	}

	var fullText string
	// Get total pages
	numPages := reader.NumPage()

	for i := 1; i <= numPages; i++ {
		page := reader.Page(i)
		if page.V.IsNull() {
			continue
		}

		text, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}

		fullText += text + "\n"
	}

	return fullText, nil
}
