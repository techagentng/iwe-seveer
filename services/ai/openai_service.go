package ai

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/sashabaranov/go-openai"
)

// OpenAIService handles interactions with OpenAI API
type OpenAIService struct {
	client *openai.Client
	model  string
}

// NewOpenAIService creates a new OpenAI service instance
func NewOpenAIService(apiKey string) *OpenAIService {
	if apiKey == "" {
		log.Println("⚠️  OpenAI API key not set - AI features will use placeholder responses")
		return &OpenAIService{
			client: nil,
			model:  openai.GPT4oMini, // GPT-4o-mini model
		}
	}

	client := openai.NewClient(apiKey)
	log.Println("✅ OpenAI service initialized with GPT-4o-mini")

	return &OpenAIService{
		client: client,
		model:  openai.GPT4oMini,
	}
}

// AnalyzeDocument analyzes document text with a user prompt
func (s *OpenAIService) AnalyzeDocument(ctx context.Context, documentText, userPrompt string) (string, error) {
	if s.client == nil {
		return s.generatePlaceholderResponse(documentText, userPrompt), nil
	}

	// Prepare system message
	systemMessage := `You are an intelligent document analysis assistant. 
Your task is to analyze documents and answer questions about them accurately and concisely.
Focus on extracting key information, patterns, and insights from the provided text.`

	// Prepare user message
	userMessage := fmt.Sprintf(`Document Content:
---
%s
---

User Question: %s

Please provide a detailed analysis addressing the user's question.`, documentText, userPrompt)

	// Create chat completion request
	resp, err := s.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: s.model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: systemMessage,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: userMessage,
				},
			},
			Temperature: 0.7,
			MaxTokens:   2000,
		},
	)

	if err != nil {
		return "", fmt.Errorf("OpenAI API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	return resp.Choices[0].Message.Content, nil
}

// AnalyzeDocumentStream analyzes document with streaming response
func (s *OpenAIService) AnalyzeDocumentStream(ctx context.Context, documentText, userPrompt string, chunkCallback func(string)) error {
	if s.client == nil {
		// Use placeholder for streaming
		response := s.generatePlaceholderResponse(documentText, userPrompt)
		s.streamPlaceholder(response, chunkCallback)
		return nil
	}

	// Prepare messages
	systemMessage := `You are an intelligent document analysis assistant. 
Your task is to analyze documents and answer questions about them accurately and concisely.
Focus on extracting key information, patterns, and insights from the provided text.`

	userMessage := fmt.Sprintf(`Document Content:
---
%s
---

User Question: %s

Please provide a detailed analysis addressing the user's question.`, documentText, userPrompt)

	// Create streaming request
	stream, err := s.client.CreateChatCompletionStream(
		ctx,
		openai.ChatCompletionRequest{
			Model: s.model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: systemMessage,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: userMessage,
				},
			},
			Temperature: 0.7,
			MaxTokens:   2000,
			Stream:      true,
		},
	)

	if err != nil {
		return fmt.Errorf("failed to create stream: %w", err)
	}
	defer stream.Close()

	// Read stream and send chunks
	for {
		response, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("stream error: %w", err)
		}

		if len(response.Choices) > 0 {
			chunk := response.Choices[0].Delta.Content
			if chunk != "" {
				chunkCallback(chunk)
			}
		}
	}

	return nil
}

// generatePlaceholderResponse generates a mock response when API key is not set
func (s *OpenAIService) generatePlaceholderResponse(documentText, userPrompt string) string {
	if userPrompt == "" {
		userPrompt = "Analyze this document"
	}

	return fmt.Sprintf(`[AI Analysis - Placeholder Mode]

Question: %s

Document Summary:
- Total characters: %d
- Estimated word count: ~%d words

Analysis:
This is a placeholder response because OpenAI API key is not configured.
To enable real AI analysis:
1. Get an API key from https://platform.openai.com/api-keys
2. Add OPENAI_API_KEY to your .env file
3. Restart the server

The document has been successfully processed and extracted.
Text extraction completed using OCR technology.

Document Preview:
%s

---
Note: Configure OPENAI_API_KEY for real GPT-4o-mini analysis.`,
		userPrompt,
		len(documentText),
		len(documentText)/5,
		s.truncateText(documentText, 500),
	)
}

// streamPlaceholder simulates streaming for placeholder response
func (s *OpenAIService) streamPlaceholder(response string, chunkCallback func(string)) {
	chunkSize := 50
	for i := 0; i < len(response); i += chunkSize {
		end := i + chunkSize
		if end > len(response) {
			end = len(response)
		}
		chunkCallback(response[i:end])
	}
}

// truncateText truncates text to a maximum length
func (s *OpenAIService) truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}

// IsConfigured returns true if OpenAI client is configured
func (s *OpenAIService) IsConfigured() bool {
	return s.client != nil
}
