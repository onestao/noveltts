package provider

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"noveltts/internal/model"
)

type MiniMaxProvider struct {
	name    string
	baseURL string
	apiKey  string
	groupID string
	client  *http.Client
}

type minimaxExtra struct {
	GroupID string `json:"group_id"`
}

func NewMiniMaxProvider(cfg model.ProviderConfig) *MiniMaxProvider {
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://api.minimaxi.com/v1"
	}

	var extra minimaxExtra
	if cfg.Extra != nil {
		json.Unmarshal(cfg.Extra, &extra)
	}

	return &MiniMaxProvider{
		name:    cfg.Name,
		baseURL: baseURL,
		apiKey:  cfg.APIKey,
		groupID: extra.GroupID,
		client:  &http.Client{},
	}
}

func (p *MiniMaxProvider) Name() string { return p.name }
func (p *MiniMaxProvider) Type() string { return "minimax" }

type minimaxT2ARequest struct {
	Model     string `json:"model"`
	Text      string `json:"text"`
	VoiceSetting struct {
		VoiceID  string  `json:"voice_id"`
		Speed    float64 `json:"speed"`
		Vol      float64 `json:"vol"`
		Pitch    int     `json:"pitch"`
	} `json:"voice_setting"`
	AudioSetting struct {
		SampleRate int    `json:"sample_rate"`
		Bitrate    int    `json:"bitrate"`
		Format     string `json:"format"`
		Channel    int    `json:"channel"`
	} `json:"audio_setting"`
}

type minimaxT2AResponse struct {
	BaseResp struct {
		StatusCode int    `json:"status_code"`
		StatusMsg  string `json:"status_msg"`
	} `json:"base_resp"`
	AudioFile string `json:"audio_file"`
}

func (p *MiniMaxProvider) Synthesize(ctx context.Context, req *model.TTSRequest) (io.ReadCloser, string, error) {
	speed := req.Speed
	if speed <= 0 {
		speed = 1.0
	}
	if speed < 0.5 {
		speed = 0.5
	}
	if speed > 2.0 {
		speed = 2.0
	}

	format := req.ResponseFormat
	if format == "" {
		format = "mp3"
	}

	modelName := req.Model
	if modelName == "" {
		modelName = "speech-01-turbo"
	}

	voiceID := req.Voice
	if voiceID == "" {
		voiceID = "male-qn-qingse"
	}

	t2aReq := minimaxT2ARequest{
		Model: modelName,
		Text:  req.Text,
	}
	t2aReq.VoiceSetting.VoiceID = voiceID
	t2aReq.VoiceSetting.Speed = speed
	t2aReq.VoiceSetting.Vol = 1.0
	t2aReq.VoiceSetting.Pitch = 0
	t2aReq.AudioSetting.SampleRate = 32000
	t2aReq.AudioSetting.Bitrate = 128000
	t2aReq.AudioSetting.Format = format
	t2aReq.AudioSetting.Channel = 1

	jsonData, err := json.Marshal(t2aReq)
	if err != nil {
		return nil, "", fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/t2a_v2?GroupId=%s", p.baseURL, p.groupID)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
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

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var t2aResp minimaxT2AResponse
	if err := json.Unmarshal(bodyBytes, &t2aResp); err != nil {
		return nil, "", fmt.Errorf("decode response: %w", err)
	}

	if t2aResp.BaseResp.StatusCode != 0 {
		return nil, "", fmt.Errorf("MiniMax error %d: %s", t2aResp.BaseResp.StatusCode, t2aResp.BaseResp.StatusMsg)
	}

	if t2aResp.AudioFile == "" {
		return nil, "", fmt.Errorf("empty audio file in response")
	}

	audioBytes, err := hex.DecodeString(t2aResp.AudioFile)
	if err != nil {
		return nil, "", fmt.Errorf("decode audio hex: %w", err)
	}

	contentType := "audio/mpeg"
	if format == "wav" {
		contentType = "audio/wav"
	} else if format == "flac" {
		contentType = "audio/flac"
	}

	return io.NopCloser(bytes.NewReader(audioBytes)), contentType, nil
}

func (p *MiniMaxProvider) ListModels(ctx context.Context) ([]model.ModelInfo, error) {
	return []model.ModelInfo{
		{ID: "speech-01-turbo", Name: "Speech-01-Turbo", Provider: p.name},
		{ID: "speech-01-hd", Name: "Speech-01-HD", Provider: p.name},
	}, nil
}

func (p *MiniMaxProvider) ListVoices(ctx context.Context, modelID string) ([]model.VoiceInfo, error) {
	return []model.VoiceInfo{
		{ID: "male-qn-qingse", Name: "青涩青年", Gender: "male"},
		{ID: "male-qn-jingying", Name: "精英青年", Gender: "male"},
		{ID: "male-qn-badao", Name: "霸道青年", Gender: "male"},
		{ID: "female-shaonv", Name: "元气少女", Gender: "female"},
		{ID: "female-yujie", Name: "知性女性", Gender: "female"},
		{ID: "female-chengshu", Name: "成熟女性", Gender: "female"},
		{ID: "presenter_male", Name: "男性主持人", Gender: "male"},
		{ID: "presenter_female", Name: "女性主持人", Gender: "female"},
	}, nil
}
