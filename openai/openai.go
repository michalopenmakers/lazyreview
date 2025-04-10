package openai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

func CodeReview(cfg *config.Config, codeChanges string, isFullReview bool) (string, error) {
	url := "https://api.openai.com/v1/chat/completions"
	var promptText string
	if isFullReview {
		promptText = "You are an experienced developer performing a complete code analysis. This is the project's first review, so analyze the project structure, code quality, potential security issues, performance and adherence to best practices. Be specific and helpful. Provide solution examples when possible."
	} else {
		promptText = "You are an experienced developer performing a code review. Your task is to find potential bugs, security vulnerabilities, performance issues, and suggest improvements to the code quality. Be specific and helpful. Provide solution examples when possible."
	}

	// Jeśli pełny review i kod długi, dzielimy na segmenty
	if isFullReview && len(codeChanges) > 1500 {
		var aggregatedReview string
		segmentSize := 1500
		segments := []string{}
		for i := 0; i < len(codeChanges); i += segmentSize {
			end := i + segmentSize
			if end > len(codeChanges) {
				end = len(codeChanges)
			}
			segments = append(segments, codeChanges[i:end])
		}
		for idx, segment := range segments {
			segPrompt := fmt.Sprintf("%s\n\nSegment %d of %d", promptText, idx+1, len(segments))
			messages := []Message{
				{Role: "system", Content: segPrompt},
				{Role: "user", Content: "Review the following code segment:\n\n" + segment},
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
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				return "", fmt.Errorf("API responded with status code: %d, body: %s", resp.StatusCode, string(body))
			}
			var response CompletionResponse
			if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
				return "", err
			}
			if len(response.Choices) == 0 {
				return "", fmt.Errorf("no response choices returned for segment %d", idx+1)
			}
			aggregatedReview += response.Choices[0].Message.Content + "\n"
		}
		return aggregatedReview, nil
	} else {
		messages := []Message{
			{Role: "system", Content: promptText},
			{Role: "user", Content: "Perform a code review for the following changes:\n\n" + codeChanges},
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
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return "", fmt.Errorf("API responded with status code: %d, body: %s", resp.StatusCode, string(body))
		}
		var response CompletionResponse
		if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return "", err
		}
		if len(response.Choices) == 0 {
			return "", fmt.Errorf("no response choices returned")
		}
		return response.Choices[0].Message.Content, nil
	}
}
