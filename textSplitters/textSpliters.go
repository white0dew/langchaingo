package textSplitters

import (
	"fmt"
	"strings"

	"github.com/tmc/langchaingo/schema"
)

type TextSplitter interface {
	SplitText(string) ([]string, error)
}

func SplitDocuments(textSplitter TextSplitter, documents []schema.Document) ([]schema.Document, error) {
	texts := make([]string, 0)
	metadatas := make([]map[string]any, 0)
	for _, document := range documents {
		texts = append(texts, document.PageContent)
		metadatas = append(metadatas, document.Metadata)
	}

	return CreateDocuments(textSplitter, texts, metadatas)
}

type LineData struct {
	From int
	To   int
}

type LOCMetadata struct {
	Lines LineData
}

// Creates documents with a text splitter
func CreateDocuments(textSplitter TextSplitter, texts []string, metadatas []map[string]any) ([]schema.Document, error) {
	if len(metadatas) == 0 {
		metadatas = make([]map[string]any, len(texts))
	}

	if len(texts) != len(metadatas) {
		return []schema.Document{}, fmt.Errorf("Number of texts does not match number of metadatas")
	}

	documents := make([]schema.Document, 0)

	for i := 0; i < len(texts); i++ {
		text := texts[i]
		lineCounterIndex := 1
		prevChunk := ""
		first := true

		chunks, err := textSplitter.SplitText(text)
		if err != nil {
			return []schema.Document{}, err
		}

		for _, chunk := range chunks {
			//Find complete number of newlines between last chunk and current chunk
			numberOfIntermediateNewLines := 0
			if !first {
				prevIndexChunk := strings.Index(text, prevChunk)
				indexChunk := strings.Index(text, chunk)

				if indexChunk == -1 || prevIndexChunk == -1 {
					return []schema.Document{}, fmt.Errorf("Error creating documents with text splitter: chunk generated by text splitter is not in the text")
				}

				completeLastChunk := string([]rune(text)[prevIndexChunk:indexChunk])
				numberOfIntermediateNewLines = strings.Count(completeLastChunk, "\n")
			}

			lineCounterIndex += numberOfIntermediateNewLines
			newLinesCount := strings.Count(chunk, "\n")

			curMetadata := make(map[string]any)
			for key, value := range metadatas[i] {
				curMetadata[key] = value
			}

			curMetadata["loc"] = LOCMetadata{
				Lines: struct {
					From int
					To   int
				}{
					From: lineCounterIndex,
					To:   lineCounterIndex + newLinesCount,
				},
			}

			documents = append(documents, schema.Document{
				PageContent: chunk,
				Metadata:    curMetadata,
			})

			first = false
			prevChunk = chunk
		}
	}

	return documents, nil
}

func joinDocs(docs []string, separator string) string {
	return strings.TrimSpace(strings.Join(docs, separator))
}

func MergeSplits(splits []string, separator string, chunkSize int, chunkOverlap int) []string {
	docs := make([]string, 0)
	currentDoc := make([]string, 0)
	total := 0

	for _, split := range splits {
		if total+len(split) > chunkSize {
			if total > chunkSize {
				fmt.Printf("Warning: created a chunk size of %v, which is longer then the specified %v", total, chunkSize)
			}

			if len(currentDoc) > 0 {
				doc := joinDocs(currentDoc, separator)
				if doc != "" {
					docs = append(docs, doc)
				}

				for total > chunkOverlap || (total+len(split) > chunkSize && total > 0) {
					total -= len(currentDoc[0])
					currentDoc = currentDoc[1:]
				}
			}
		}

		currentDoc = append(currentDoc, split)
		total += len(split)
	}

	doc := joinDocs(currentDoc, separator)
	if doc != "" {
		docs = append(docs, doc)
	}

	return docs
}
