package openai

import (
	"context"
	"fmt"

	openai "github.com/sashabaranov/go-openai"
)

type ChatClient struct {
	client *openai.Client
	model  string
}

func NewChatClient(apiKey, model string) *ChatClient {
	return &ChatClient{
		client: openai.NewClient(apiKey),
		model:  model,
	}
}

// generate answer
func (c *ChatClient) GenerateAnswer(
	ctx context.Context,
	query string,
	context string,
) (string, error) {
	systemPrompt := `Anda adalah asisten AI yang membantu menjawab pertanyaan berdasarkan dokumen yang diberikan.
	
	Instruksi:
	1. Jawab pertanyaan HANYA berdasarkan konteks yang diberikan
	2. Jika informasi tidak ada dalam konteks, katakan "Maaf, saya tidak menemukan informasi tersebut dalam dokumen"
	3. Berikan jawaban yang jelas, ringkas, dan terstruktur
	4. Gunakan bahasa Indonesia yang baik dan benar`

	userPrompt := fmt.Sprintf(`Konteks dari dokumen:
	%s

	Pertanyaan: %s

	Jawaban:`, context, query)

	resp, err := c.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: c.model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: systemPrompt,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: userPrompt,
			},
		},
		Temperature: 0.7,
		MaxTokens:   700,
	})

	if err != nil {
		return "", fmt.Errorf("Failed to generate answer: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAi")
	}

	return resp.Choices[0].Message.Content, nil
}
