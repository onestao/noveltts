package provider

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"noveltts/internal/model"
)

type MiMoProvider struct {
	name    string
	baseURL string
	apiKey  string
	client  *http.Client
}

func NewMiMoProvider(cfg model.ProviderConfig) *MiMoProvider {
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://api.xiaomimimo.com/v1"
	}
	return &MiMoProvider{
		name:    cfg.Name,
		baseURL: baseURL,
		apiKey:  cfg.APIKey,
		client:  &http.Client{},
	}
}

func (p *MiMoProvider) Name() string { return p.name }
func (p *MiMoProvider) Type() string { return "mimo" }

type mimoChatRequest struct {
	Model    string        `json:"model"`
	Messages []mimoMessage `json:"messages"`
	Audio    *mimoAudio    `json:"audio,omitempty"`
	Stream   bool          `json:"stream"`
}

type mimoMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type mimoAudio struct {
	Format string `json:"format,omitempty"`
	Voice  string `json:"voice,omitempty"`
}

type mimoChatResponse struct {
	Choices []struct {
		Message struct {
			Audio struct {
				Data string `json:"data"`
			} `json:"audio"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (p *MiMoProvider) Synthesize(ctx context.Context, req *model.TTSRequest) (io.ReadCloser, string, error) {
	modelName := req.Model
	if modelName == "" {
		modelName = "mimo-v2.5-tts"
	}

	voice := req.Voice
	if voice == "" {
		voice = "mimo_default"
	}

	format := req.ResponseFormat
	if format == "" {
		format = "mp3"
	}

	messages := []mimoMessage{
		{Role: "assistant", Content: req.Text},
	}

	if strings.Contains(modelName, "voicedesign") {
		messages = append([]mimoMessage{
			{Role: "user", Content: "请用指定的声音朗读以下文本"},
		}, messages...)
	}

	body := mimoChatRequest{
		Model:    modelName,
		Messages: messages,
		Audio: &mimoAudio{
			Format: format,
			Voice:  voice,
		},
		Stream: false,
	}

	jsonData, err := json.Marshal(body)
	if err != nil {
		return nil, "", fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(jsonData))
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
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBytes))
	}

	var mimoResp mimoChatResponse
	if err := json.Unmarshal(respBytes, &mimoResp); err != nil {
		return nil, "", fmt.Errorf("decode response: %w", err)
	}

	if mimoResp.Error != nil {
		return nil, "", fmt.Errorf("MiMo error: %s", mimoResp.Error.Message)
	}

	if len(mimoResp.Choices) == 0 {
		return nil, "", fmt.Errorf("empty choices in response")
	}

	audioData := mimoResp.Choices[0].Message.Audio.Data
	if audioData == "" {
		return nil, "", fmt.Errorf("empty audio data in response")
	}

	audioBytes, err := base64.StdEncoding.DecodeString(audioData)
	if err != nil {
		return nil, "", fmt.Errorf("decode audio base64: %w", err)
	}

	contentType := "audio/mpeg"
	switch format {
	case "wav":
		contentType = "audio/wav"
	case "pcm", "pcm16":
		contentType = "audio/pcm"
	}

	return io.NopCloser(bytes.NewReader(audioBytes)), contentType, nil
}

func (p *MiMoProvider) ListModels(ctx context.Context) ([]model.ModelInfo, error) {
	return []model.ModelInfo{
		{ID: "mimo-v2.5-tts", Name: "MiMo V2.5 TTS", Provider: p.name},
		{ID: "mimo-v2.5-tts-voicedesign", Name: "MiMo V2.5 TTS VoiceDesign", Provider: p.name},
		{ID: "mimo-v2.5-tts-voiceclone", Name: "MiMo V2.5 TTS VoiceClone", Provider: p.name},
		{ID: "mimo-v2-tts", Name: "MiMo V2 TTS", Provider: p.name},
	}, nil
}

func (p *MiMoProvider) ListVoices(ctx context.Context, modelID string) ([]model.VoiceInfo, error) {
	if strings.Contains(modelID, "v2.5") {
		return []model.VoiceInfo{
			{ID: "mimo_default", Name: "MiMo 默认"},
			{ID: "冰糖", Name: "冰糖 (女声)"},
			{ID: "茉莉", Name: "茉莉 (女声)"},
			{ID: "苏打", Name: "苏打 (女声)"},
			{ID: "白桦", Name: "白桦 (女声)"},
			{ID: "Mia", Name: "Mia (女声)"},
			{ID: "Chloe", Name: "Chloe (女声)"},
			{ID: "Milo", Name: "Milo (男声)"},
			{ID: "Dean", Name: "Dean (男声)"},
		}, nil
	}
	return []model.VoiceInfo{
		{ID: "mimo_default", Name: "MiMo 默认"},
		{ID: "default_en", Name: "英文默认"},
		{ID: "default_zh", Name: "中文默认"},
	}, nil
}
