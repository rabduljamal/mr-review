// services/groq_service.go
package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
)

type GroqService struct {
	apiKey     string
	httpClient *http.Client
}

type groqRequest struct {
	Model     string        `json:"model"`
	Messages  []groqMessage `json:"messages"`
	MaxTokens int          `json:"max_tokens"`
}

type groqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type groqResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func NewGroqService(apiKey string) *GroqService {
	return &GroqService{
		apiKey:     apiKey,
		httpClient: &http.Client{},
	}
}

func removeThinkTags(input string) string {
	re := regexp.MustCompile(`<think>.*?</think>`)
	return re.ReplaceAllString(input, "")
}

func (s *GroqService) GenerateReview(prompt string) (string, error) {
	reqBody := groqRequest{
		Model: "deepseek-r1-distill-llama-70b",
		Messages: []groqMessage{
			{
				Role:    "system",
				Content: "You are a senior software engineer reviewing code changes. Provide detailed, constructive feedback focusing on best practices, security, and performance.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens: 2000,
	}

	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	req, err := http.NewRequest("POST", "https://api.groq.com/openai/v1/chat/completions", bytes.NewBuffer(reqJSON))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var groqResp groqResponse
	if err := json.Unmarshal(body, &groqResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	if len(groqResp.Choices) == 0 {
		return "", fmt.Errorf("no response from Groq API")
	}

	return removeThinkTags(groqResp.Choices[0].Message.Content), nil
}