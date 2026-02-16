package document

import (
	"strings"
	"unicode"
)

type Chunker struct {
	chunkSize    int
	chunkOverlap int
}

// NewChunker creates a new chunker
func NewChunker(chunkSize, chunkOverlap int) *Chunker {
	return &Chunker{
		chunkSize:    chunkSize,
		chunkOverlap: chunkOverlap,
	}
}

func (c *Chunker) ChunkText(text string) []string {
	//clean text
	text = strings.TrimSpace(text)
	text = cleanText(text)

	if len(text) == 0 {
		return []string{}
	}

	var chunks []string
	start := 0

	for start < len(text) {
		end := start + c.chunkSize
		if end > len(text) {
			end = len(text)
		}

		// try to break at sentence boundary
		if end < len(text) {
			for i := end; i > start+c.chunkSize/2; i-- {
				if text[i] == '.' || text[i] == '!' || text[i] == '?' || text[i] == '\n' {
					end = i + 1
					break
				}
			}
		}
		chunk := strings.TrimSpace(text[start:end])
		if len(chunk) > 0 {
			chunks = append(chunks, chunk)
		}

		// move start position with overlap
		start = end - c.chunkOverlap
		if start < 0 {
			start = 0
		}
	}

	return chunks
}

func cleanText(text string) string {
	// remove multiple whitespace
	var result strings.Builder
	prevSpace := false

	for _, r := range text {
		if unicode.IsSpace(r) {
			if !prevSpace {
				result.WriteRune(' ')
				prevSpace = true
			}
		} else {
			result.WriteRune(r)
			prevSpace = false
		}

	}

	return result.String()
}
