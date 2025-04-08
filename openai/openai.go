package openai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/michalopenmakers/lazyreview/config"
)

type CompletionRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens"`
	Temperature float64   `json:"temperature"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type CompletionResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func Query(cfg *config.Config, prompt string) string {
	return "Response for: " + prompt
}

func CodeReview(cfg *config.Config, codeChanges string) (string, error) {
	url := "https://api.openai.com/v1/chat/completions"
	isFullRepo := strings.Contains(codeChanges, "Entire project code")
	var promptText string
	if isFullRepo {
		promptText = "You are an experienced developer performing a complete code analysis. This is the project's first review, so analyze the project structure, code quality, potential security issues, performance and adherence to best practices. Pay attention to the architecture and overall code organization. Be specific and helpful. Provide solution examples when possible."
	} else {
		promptText = "You are an experienced developer performing a code review. Your task is to find potential bugs, security vulnerabilities, performance issues, and suggest improvements to the code quality. Be specific and helpful. Provide solution examples when possible."
	}

	const maxInputLength = 2000
	if len(codeChanges) > maxInputLength {
		codeChanges = codeChanges[:maxInputLength] + "\n... [Content truncated due to token limits]"
	}

	messages := []Message{
		{
			Role:    "system",
			Content: promptText,
		},
		{
			Role:    "user",
			Content: "Perform a code review for the following changes:\n\n" + codeChanges,
		},
	}
	requestBody, err := json.Marshal(CompletionRequest{
		Model:       cfg.AIModelConfig.Model,
		Messages:    messages,
		MaxTokens:   cfg.AIModelConfig.MaxTokens,
		Temperature: 0.5,
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.AIModelConfig.ApiKey)
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println("Error closing response body:", err)
		}
	}(resp.Body)
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API responded with status code: %d, body: %s", resp.StatusCode, string(body))
	}
	var response CompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", err
	}
	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned")
	}
	return response.Choices[0].Message.Content, nil
}
