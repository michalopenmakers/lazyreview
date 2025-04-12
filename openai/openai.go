package openai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/michalopenmakers/lazyreview/config"
	"github.com/michalopenmakers/lazyreview/logger"
)

type CompletionRequest struct {
	Model               string    `json:"model"`
	Messages            []Message `json:"messages"`
	MaxCompletionTokens int       `json:"max_completion_tokens"`
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
	logger.Log("Starting CodeReview request")
	url := "https://api.openai.com/v1/chat/completions"
	var promptText string
	if isFullReview {
		promptText = "You are an experienced developer performing a complete code analysis. This is the project's first review, so analyze the project structure, code quality, potential security issues, performance and adherence to best practices. Be specific and helpful. Provide solution examples when possible."
	} else {
		promptText = "You are an experienced developer performing a code review. Your task is to find potential bugs, security vulnerabilities, performance issues, and suggest improvements to the code quality. Be specific and helpful. Provide solution examples when possible."
	}
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
			logger.Log(fmt.Sprintf("Sending API request for segment %d of %d", idx+1, len(segments)))
			requestBody, err := json.Marshal(CompletionRequest{
				Model:               cfg.AIModelConfig.Model,
				Messages:            messages,
				MaxCompletionTokens: cfg.AIModelConfig.MaxTokens,
			})
			if err != nil {
				logger.Log(fmt.Sprintf("Error marshaling request: %v", err))
				return "", err
			}
			req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
			if err != nil {
				logger.Log(fmt.Sprintf("Error creating request: %v", err))
				return "", err
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+cfg.AIModelConfig.ApiKey)
			client := &http.Client{Timeout: 60 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				logger.Log(fmt.Sprintf("HTTP request error: %v", err))
				return "", err
			}
			defer func(Body io.ReadCloser) {
				err := Body.Close()
				if err != nil {

				}
			}(resp.Body)
			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				errMsg := fmt.Sprintf("API responded with status code: %d, body: %s", resp.StatusCode, string(body))
				logger.Log(errMsg)
				return "", fmt.Errorf(errMsg)
			}
			var response CompletionResponse
			if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
				logger.Log(fmt.Sprintf("Error decoding response: %v", err))
				return "", err
			}
			if len(response.Choices) == 0 {
				errMsg := fmt.Sprintf("No response choices returned for segment %d", idx+1)
				logger.Log(errMsg)
				return "", fmt.Errorf(errMsg)
			}
			logger.Log(fmt.Sprintf("Received response for segment %d", idx+1))
			aggregatedReview += response.Choices[0].Message.Content + "\n"
		}
		return aggregatedReview, nil
	} else {
		messages := []Message{
			{Role: "system", Content: promptText},
			{Role: "user", Content: "Perform a code review for the following changes:\n\n" + codeChanges},
		}
		logger.Log("Sending API request for full review or changes")
		requestBody, err := json.Marshal(CompletionRequest{
			Model:               cfg.AIModelConfig.Model,
			Messages:            messages,
			MaxCompletionTokens: cfg.AIModelConfig.MaxTokens,
		})
		if err != nil {
			logger.Log(fmt.Sprintf("Error marshaling request: %v", err))
			return "", err
		}
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
		if err != nil {
			logger.Log(fmt.Sprintf("Error creating request: %v", err))
			return "", err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+cfg.AIModelConfig.ApiKey)
		client := &http.Client{Timeout: 60 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			logger.Log(fmt.Sprintf("HTTP request error: %v", err))
			return "", err
		}
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {

			}
		}(resp.Body)
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			errMsg := fmt.Sprintf("API responded with status code: %d, body: %s", resp.StatusCode, string(body))
			logger.Log(errMsg)
			return "", fmt.Errorf(errMsg)
		}
		var response CompletionResponse
		if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
			logger.Log(fmt.Sprintf("Error decoding response: %v", err))
			return "", err
		}
		if len(response.Choices) == 0 {
			errMsg := "No response choices returned"
			logger.Log(errMsg)
			return "", fmt.Errorf(errMsg)
		}
		logger.Log("Received API response for code review")
		return response.Choices[0].Message.Content, nil
	}
}
