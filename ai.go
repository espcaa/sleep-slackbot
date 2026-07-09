package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type AiClient struct {
	BaseUrl string
	ApiKey  string
	AiModel string
}

func NewAiClient(baseUrl, apiKey, aiModel string) *AiClient {
	return &AiClient{
		BaseUrl: baseUrl,
		ApiKey:  apiKey,
		AiModel: aiModel,
	}
}

type AiResponse struct {
	Choices []struct {
		Message AiMessage `json:"message"`
	} `json:"choices"`
}

type AiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AiRequest struct {
	Messages []AiMessage `json:"messages"`
	Model    string      `json:"model"`
}

func (m *AiResponse) GetContent() string {
	if len(m.Choices) > 0 {
		return m.Choices[0].Message.Content
	}
	return ""
}

func (c *AiClient) Complete(messages []AiMessage) (AiResponse, error) {
	// make the request to the ai api
	client := &http.Client{}
	req, err := http.NewRequest("POST", c.BaseUrl, nil)
	if err != nil {
		return AiResponse{}, err
	}

	// set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.ApiKey)

	aiRequest := AiRequest{
		Messages: messages,
		Model:    c.AiModel,
	}

	// set body
	body, err := json.Marshal(aiRequest)
	if err != nil {
		return AiResponse{}, err
	}
	req.Body = io.NopCloser(bytes.NewReader(body))

	// send the request
	resp, err := client.Do(req)
	if err != nil {
		return AiResponse{}, err
	}
	defer resp.Body.Close()

	// read the response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return AiResponse{}, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return AiResponse{}, fmt.Errorf("api error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var aiResp AiResponse
	err = json.Unmarshal(respBody, &aiResp)
	if err != nil {
		return AiResponse{}, fmt.Errorf("json unmarshal failed: %w (raw response: %s)", err, string(respBody))
	}

	return aiResp, nil
}
