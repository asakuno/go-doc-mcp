package document

import (
	"strings"
	"unicode/utf8"
)

type Chunker struct {
	chunkSize    int
	chunkOverlap int
}

type Chunk struct {
	Text     string
	Index    int
	Document *Document
}

func NewChunker(chunkSize, chunkOverlap int) *Chunker {
	return &Chunker{
		chunkSize:    chunkSize,
		chunkOverlap: chunkOverlap,
	}
}

// ChunkDocument splits a document into overlapping chunks
func (c *Chunker) ChunkDocument(doc *Document) []*Chunk {
	text := doc.Content
	if utf8.RuneCountInString(text) <= c.chunkSize {
		return []*Chunk{{
			Text:     text,
			Index:    0,
			Document: doc,
		}}
	}

	var chunks []*Chunk
	runes := []rune(text)
	start := 0
	index := 0

	for start < len(runes) {
		end := start + c.chunkSize
		if end > len(runes) {
			end = len(runes)
		}

		// Try to break at sentence boundary
		if end < len(runes) {
			// Look for sentence ending punctuation
			breakPoint := c.findBreakPoint(runes[start:end])
			if breakPoint > 0 {
				end = start + breakPoint
			}
		}

		chunkText := string(runes[start:end])
		chunks = append(chunks, &Chunk{
			Text:     strings.TrimSpace(chunkText),
			Index:    index,
			Document: doc,
		})

		index++
		start = end - c.chunkOverlap
		if start < 0 {
			start = 0
		}
	}

	return chunks
}

// findBreakPoint finds a good breaking point (sentence end, paragraph, etc.)
func (c *Chunker) findBreakPoint(runes []rune) int {
	sentenceEndings := []rune{'.', '!', '?', '\n'}

	// Search backwards from the end
	for i := len(runes) - 1; i >= len(runes)-200 && i >= 0; i-- {
		for _, ending := range sentenceEndings {
			if runes[i] == ending {
				// Check if followed by space or newline
				if i+1 < len(runes) && (runes[i+1] == ' ' || runes[i+1] == '\n') {
					return i + 1
				}
			}
		}
	}

	return -1 // No good break point found
}

// ChunkDocuments chunks multiple documents
func (c *Chunker) ChunkDocuments(docs []*Document) []*Chunk {
	var allChunks []*Chunk
	for _, doc := range docs {
		chunks := c.ChunkDocument(doc)
		allChunks = append(allChunks, chunks...)
	}
	return allChunks
}
