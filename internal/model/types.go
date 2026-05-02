package model

import "encoding/json"

type TTSRequest struct {
	Text           string  `json:"text"`
	Voice          string  `json:"voice"`
	Speed          float64 `json:"speed"`
	ResponseFormat string  `json:"response_format"`
	Model          string  `json:"model"`
	Provider       string  `json:"provider"`
}

type ModelInfo struct {
	ID       string      `json:"id"`
	Name     string      `json:"name"`
	Provider string      `json:"provider"`
	Voices   []VoiceInfo `json:"voices,omitempty"`
}

type VoiceInfo struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	PreviewURL string `json:"preview_url,omitempty"`
	Gender     string `json:"gender,omitempty"`
}

type ProviderConfig struct {
	Name          string          `json:"name"`
	Type          string          `json:"type"`
	BaseURL       string          `json:"base_url"`
	APIKey        string          `json:"api_key"`
	DefaultModel  string          `json:"default_model"`
	DefaultVoice  string          `json:"default_voice"`
	Extra         json.RawMessage `json:"extra,omitempty"`
}

type ServerConfig struct {
	Port     int    `json:"port"`
	LogLevel string `json:"log_level"`
	AuthToken string `json:"auth_token,omitempty"`
}

type DefaultsConfig struct {
	Provider string  `json:"provider"`
	Speed    float64 `json:"speed"`
	Format   string  `json:"format"`
}

type AppConfig struct {
	Server    ServerConfig     `json:"server"`
	Providers []ProviderConfig `json:"providers"`
	Defaults  DefaultsConfig   `json:"defaults"`
}

type OpenAIRequest struct {
	Model          string  `json:"model"`
	Input          string  `json:"input"`
	Voice          string  `json:"voice"`
	Speed          float64 `json:"speed"`
	ResponseFormat string  `json:"response_format"`
}

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code,omitempty"`
}
