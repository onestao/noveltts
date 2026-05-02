package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"noveltts/internal/model"
)

type OpenAIProvider struct {
	name    string
	baseURL string
	apiKey  string
	client  *http.Client
}

func NewOpenAIProvider(cfg model.ProviderConfig) *OpenAIProvider {
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if !strings.HasSuffix(baseURL, "/v1") && !strings.HasSuffix(baseURL, "/v1/") {
		if strings.Contains(baseURL, "openai.com") || strings.Contains(baseURL, "siliconflow") ||
			strings.Contains(baseURL, "xiaomimimo") || strings.Contains(baseURL, "api.") {
			baseURL += "/v1"
		}
	}
	return &OpenAIProvider{
		name:    cfg.Name,
		baseURL: baseURL,
		apiKey:  cfg.APIKey,
		client:  &http.Client{},
	}
}

func (p *OpenAIProvider) Name() string { return p.name }
func (p *OpenAIProvider) Type() string { return "openai_compatible" }

type openAISpeechRequest struct {
	Model          string  `json:"model"`
	Input          string  `json:"input"`
	Voice          string  `json:"voice"`
	Speed          float64 `json:"speed"`
	ResponseFormat string  `json:"response_format,omitempty"`
}

func (p *OpenAIProvider) Synthesize(ctx context.Context, req *model.TTSRequest) (io.ReadCloser, string, error) {
	speed := req.Speed
	if speed <= 0 {
		speed = 1.0
	}
	if speed < 0.25 {
		speed = 0.25
	}
	if speed > 4.0 {
		speed = 4.0
	}

	format := req.ResponseFormat
	if format == "" {
		format = "mp3"
	}

	body := openAISpeechRequest{
		Model:          req.Model,
		Input:          req.Text,
		Voice:          req.Voice,
		Speed:          speed,
		ResponseFormat: format,
	}

	jsonData, err := json.Marshal(body)
	if err != nil {
		return nil, "", fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/audio/speech", bytes.NewReader(jsonData))
	if err != nil {
		return nil, "", fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, "", fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		errBody, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(errBody))
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "audio/mpeg"
	}

	return resp.Body, contentType, nil
}

func (p *OpenAIProvider) ListModels(ctx context.Context) ([]model.ModelInfo, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/models", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	if p.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(errBody))
	}

	var result struct {
		Data []struct {
			ID      string `json:"id"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	var models []model.ModelInfo
	for _, m := range result.Data {
		models = append(models, model.ModelInfo{
			ID:       m.ID,
			Name:     m.ID,
			Provider: p.name,
		})
	}
	return models, nil
}

func (p *OpenAIProvider) ListVoices(ctx context.Context, modelID string) ([]model.VoiceInfo, error) {
	if strings.Contains(p.baseURL, "siliconflow") {
		return []model.VoiceInfo{
			{ID: "FunAudioLLM/CosyVoice2-0.5B:alex", Name: "alex - 沉稳男声", Gender: "male"},
			{ID: "FunAudioLLM/CosyVoice2-0.5B:benjamin", Name: "benjamin - 低沉男声", Gender: "male"},
			{ID: "FunAudioLLM/CosyVoice2-0.5B:charles", Name: "charles - 磁性男声", Gender: "male"},
			{ID: "FunAudioLLM/CosyVoice2-0.5B:david", Name: "david - 欢快男声", Gender: "male"},
			{ID: "FunAudioLLM/CosyVoice2-0.5B:anna", Name: "anna - 沉稳女声", Gender: "female"},
			{ID: "FunAudioLLM/CosyVoice2-0.5B:bella", Name: "bella - 激情女声", Gender: "female"},
			{ID: "FunAudioLLM/CosyVoice2-0.5B:claire", Name: "claire - 温柔女声", Gender: "female"},
			{ID: "FunAudioLLM/CosyVoice2-0.5B:diana", Name: "diana - 欢快女声", Gender: "female"},
		}, nil
	}
	return []model.VoiceInfo{
		{ID: "alloy", Name: "Alloy"},
		{ID: "echo", Name: "Echo"},
		{ID: "fable", Name: "Fable"},
		{ID: "onyx", Name: "Onyx"},
		{ID: "nova", Name: "Nova"},
		{ID: "shimmer", Name: "Shimmer"},
	}, nil
}
