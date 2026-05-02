package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"noveltts/internal/config"
	"noveltts/internal/model"
	"noveltts/internal/provider"
)

func getAppConfig(c *gin.Context) *model.AppConfig {
	return config.Get()
}

type ConfigHandler struct {
	registry *provider.Registry
}

func NewConfigHandler(reg *provider.Registry) *ConfigHandler {
	return &ConfigHandler{registry: reg}
}

func (h *ConfigHandler) GetConfig(c *gin.Context) {
	cfg := config.Get()
	sanitized := *cfg
	sanitized.Providers = make([]model.ProviderConfig, len(cfg.Providers))
	for i, p := range cfg.Providers {
		sanitized.Providers[i] = p
		if sanitized.Providers[i].APIKey != "" {
			key := p.APIKey
			if len(key) > 8 {
				sanitized.Providers[i].APIKey = key[:4] + "****" + key[len(key)-4:]
			} else {
				sanitized.Providers[i].APIKey = "****"
			}
		}
	}
	c.JSON(http.StatusOK, sanitized)
}

func (h *ConfigHandler) UpdateConfig(c *gin.Context) {
	var newCfg model.AppConfig
	if err := c.ShouldBindJSON(&newCfg); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.ErrorDetail{Message: "invalid config: " + err.Error(), Type: "invalid_request_error"},
		})
		return
	}

	oldCfg := config.Get()
	oldKeys := make(map[string]string)
	for _, p := range oldCfg.Providers {
		oldKeys[p.Name] = p.APIKey
	}
	for i, p := range newCfg.Providers {
		if p.APIKey == "" || strings.Contains(p.APIKey, "****") {
			if k, ok := oldKeys[p.Name]; ok {
				newCfg.Providers[i].APIKey = k
			}
		}
	}

	if err := config.Save(&newCfg); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error: model.ErrorDetail{Message: "save config: " + err.Error(), Type: "server_error"},
		})
		return
	}

	h.registry.LoadFromConfig(&newCfg)
	c.JSON(http.StatusOK, gin.H{"message": "config saved"})
}

func (h *ConfigHandler) AddProvider(c *gin.Context) {
	var pc model.ProviderConfig
	if err := c.ShouldBindJSON(&pc); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.ErrorDetail{Message: "invalid provider config", Type: "invalid_request_error"},
		})
		return
	}

	if pc.Type == "" || pc.Type == "openai_compatible" {
		u := strings.ToLower(pc.BaseURL)
		if strings.Contains(u, "xiaomimimo") {
			pc.Type = "mimo"
		} else if strings.Contains(u, "minimax") || strings.Contains(u, "minimaxi") {
			pc.Type = "minimax"
		} else if strings.Contains(u, "doubao") || strings.Contains(u, "amantha") {
			pc.Type = "doubao"
		} else {
			pc.Type = "openai_compatible"
		}
	}

	cfg := config.Get()
	for i, p := range cfg.Providers {
		if p.Name == pc.Name {
			if pc.APIKey == "" || strings.Contains(pc.APIKey, "****") {
				pc.APIKey = p.APIKey
			}
			cfg.Providers[i] = pc
			if err := config.Save(cfg); err != nil {
				c.JSON(http.StatusInternalServerError, model.ErrorResponse{
					Error: model.ErrorDetail{Message: err.Error(), Type: "server_error"},
				})
				return
			}
			h.registry.LoadFromConfig(cfg)
			c.JSON(http.StatusOK, gin.H{"message": "provider updated"})
			return
		}
	}

	cfg.Providers = append(cfg.Providers, pc)
	if err := config.Save(cfg); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error: model.ErrorDetail{Message: err.Error(), Type: "server_error"},
		})
		return
	}
	h.registry.LoadFromConfig(cfg)
	c.JSON(http.StatusCreated, gin.H{"message": "provider added"})
}

func (h *ConfigHandler) DeleteProvider(c *gin.Context) {
	name := c.Param("name")
	cfg := config.Get()

	for i, p := range cfg.Providers {
		if p.Name == name {
			cfg.Providers = append(cfg.Providers[:i], cfg.Providers[i+1:]...)
			if err := config.Save(cfg); err != nil {
				c.JSON(http.StatusInternalServerError, model.ErrorResponse{
					Error: model.ErrorDetail{Message: err.Error(), Type: "server_error"},
				})
				return
			}
			h.registry.LoadFromConfig(cfg)
			c.JSON(http.StatusOK, gin.H{"message": "provider deleted"})
			return
		}
	}

	c.JSON(http.StatusNotFound, model.ErrorResponse{
		Error: model.ErrorDetail{Message: "provider not found", Type: "not_found_error"},
	})
}

func (h *ConfigHandler) UpdateDefaults(c *gin.Context) {
	var defaults model.DefaultsConfig
	if err := c.ShouldBindJSON(&defaults); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.ErrorDetail{Message: "invalid defaults", Type: "invalid_request_error"},
		})
		return
	}

	cfg := config.Get()
	cfg.Defaults = defaults
	if err := config.Save(cfg); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error: model.ErrorDetail{Message: err.Error(), Type: "server_error"},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "defaults updated"})
}

type previewRequest struct {
	Provider string `json:"provider"`
	Voice    string `json:"voice"`
	Model    string `json:"model"`
	Text     string `json:"text"`
	Format   string `json:"format"`
}

func (h *ConfigHandler) PreviewVoice(c *gin.Context) {
	var req previewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.ErrorDetail{Message: "invalid request", Type: "invalid_request_error"},
		})
		return
	}

	if req.Text == "" {
		req.Text = "这是一段测试语音，用于试听音色效果。"
	}

	cfg := config.Get()

	prov, ok := h.registry.Get(req.Provider)
	if !ok {
		var err error
		prov, err = h.registry.GetDefault(cfg)
		if err != nil {
			c.JSON(http.StatusBadRequest, model.ErrorResponse{
				Error: model.ErrorDetail{Message: "no provider available", Type: "invalid_request_error"},
			})
			return
		}
	}

	if req.Model == "" {
		for _, pc := range cfg.Providers {
			if pc.Name == req.Provider {
				req.Model = pc.DefaultModel
				break
			}
		}
	}

	format := req.Format
	if format == "" {
		format = "mp3"
	}

	ttsReq := &model.TTSRequest{
		Text:           req.Text,
		Voice:          req.Voice,
		Speed:          1.0,
		ResponseFormat: format,
		Model:          req.Model,
	}

	if req.Provider != "" {
		for _, pc := range cfg.Providers {
			if pc.Name == req.Provider && pc.Extra != nil {
				var extra struct {
					Style      string `json:"style"`
					Dialect    string `json:"dialect"`
					UserMessage string `json:"user_message"`
				}
				if json.Unmarshal(pc.Extra, &extra) == nil {
					if extra.Style != "" {
						ttsReq.Style = extra.Style
					}
					if extra.Dialect != "" {
						ttsReq.Dialect = extra.Dialect
					}
					if extra.UserMessage != "" {
						ttsReq.UserMessage = extra.UserMessage
					}
				}
				break
			}
		}
	}

	log.Printf("[preview] provider=%s model=%s voice=%s", req.Provider, req.Model, req.Voice)

	body, contentType, err := prov.Synthesize(c.Request.Context(), ttsReq)
	if err != nil {
		log.Printf("[preview] error: %v", err)
		c.JSON(http.StatusBadGateway, model.ErrorResponse{
			Error: model.ErrorDetail{Message: "synthesize failed: " + err.Error(), Type: "api_error"},
		})
		return
	}
	defer body.Close()

	c.Header("Content-Type", contentType)
	c.Status(http.StatusOK)
	io.Copy(c.Writer, body)
}

type fetchProviderRequest struct {
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
	Type    string `json:"type"`
}

func (h *ConfigHandler) FetchModels(c *gin.Context) {
	var req fetchProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.ErrorDetail{Message: "invalid request", Type: "invalid_request_error"},
		})
		return
	}

	baseURL := strings.TrimRight(req.BaseURL, "/")

	httpReq, err := http.NewRequestWithContext(c.Request.Context(), "GET", baseURL+"/models", nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.ErrorDetail{Message: "invalid base_url: " + err.Error(), Type: "invalid_request_error"},
		})
		return
	}
	if req.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+req.APIKey)
	}

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		c.JSON(http.StatusBadGateway, model.ErrorResponse{
			Error: model.ErrorDetail{Message: "request failed: " + err.Error(), Type: "api_error"},
		})
		return
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error: model.ErrorDetail{Message: "read response: " + err.Error(), Type: "server_error"},
		})
		return
	}

	if resp.StatusCode != http.StatusOK {
		c.JSON(resp.StatusCode, model.ErrorResponse{
			Error: model.ErrorDetail{
				Message: fmt.Sprintf("upstream error %d: %s", resp.StatusCode, string(bodyBytes)),
				Type:    "api_error",
			},
		})
		return
	}

	var result struct {
		Data []struct {
			ID      string `json:"id"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		c.JSON(http.StatusBadGateway, model.ErrorResponse{
			Error: model.ErrorDetail{Message: "decode response: " + err.Error(), Type: "api_error"},
		})
		return
	}

	var models []map[string]string
	for _, m := range result.Data {
		models = append(models, map[string]string{
			"id":   m.ID,
			"name": m.ID,
		})
	}

	c.JSON(http.StatusOK, gin.H{"data": models})
}

func (h *ConfigHandler) FetchVoices(c *gin.Context) {
	var req fetchProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.ErrorDetail{Message: "invalid request", Type: "invalid_request_error"},
		})
		return
	}

	baseURL := strings.TrimRight(req.BaseURL, "/")

	type voiceItem struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Gender string `json:"gender,omitempty"`
	}

	var voices []voiceItem

	siliconflowPresets := []voiceItem{
		{ID: "FunAudioLLM/CosyVoice2-0.5B:alex", Name: "alex - 沉稳男声", Gender: "male"},
		{ID: "FunAudioLLM/CosyVoice2-0.5B:benjamin", Name: "benjamin - 低沉男声", Gender: "male"},
		{ID: "FunAudioLLM/CosyVoice2-0.5B:charles", Name: "charles - 磁性男声", Gender: "male"},
		{ID: "FunAudioLLM/CosyVoice2-0.5B:david", Name: "david - 欢快男声", Gender: "male"},
		{ID: "FunAudioLLM/CosyVoice2-0.5B:anna", Name: "anna - 沉稳女声", Gender: "female"},
		{ID: "FunAudioLLM/CosyVoice2-0.5B:bella", Name: "bella - 激情女声", Gender: "female"},
		{ID: "FunAudioLLM/CosyVoice2-0.5B:claire", Name: "claire - 温柔女声", Gender: "female"},
		{ID: "FunAudioLLM/CosyVoice2-0.5B:diana", Name: "diana - 欢快女声", Gender: "female"},
	}

	openaiPresets := []voiceItem{
		{ID: "alloy", Name: "Alloy"},
		{ID: "echo", Name: "Echo"},
		{ID: "fable", Name: "Fable"},
		{ID: "onyx", Name: "Onyx"},
		{ID: "nova", Name: "Nova"},
		{ID: "shimmer", Name: "Shimmer"},
	}

	mimoPresets := []voiceItem{
		{ID: "mimo_default", Name: "MiMo 默认"},
		{ID: "冰糖", Name: "冰糖 (女声)"},
		{ID: "茉莉", Name: "茉莉 (女声)"},
		{ID: "苏打", Name: "苏打 (女声)"},
		{ID: "白桦", Name: "白桦 (女声)"},
		{ID: "Mia", Name: "Mia (女声)"},
		{ID: "Chloe", Name: "Chloe (女声)"},
		{ID: "Milo", Name: "Milo (男声)"},
		{ID: "Dean", Name: "Dean (男声)"},
	}

	isSiliconFlow := strings.Contains(baseURL, "siliconflow")
	isOpenAI := strings.Contains(baseURL, "openai.com")
	isMiMo := strings.Contains(baseURL, "xiaomimimo")

	if isSiliconFlow {
		voices = append(voices, siliconflowPresets...)
	} else if isOpenAI {
		voices = append(voices, openaiPresets...)
	} else if isMiMo {
		voices = append(voices, mimoPresets...)
	}

	httpReq, err := http.NewRequestWithContext(c.Request.Context(), "GET", baseURL+"/audio/voice/list", nil)
	if err == nil {
		if req.APIKey != "" {
			httpReq.Header.Set("Authorization", "Bearer "+req.APIKey)
		}
		client := &http.Client{}
		resp, err := client.Do(httpReq)
		if err == nil {
			defer resp.Body.Close()
			bodyBytes, err := io.ReadAll(resp.Body)
			if err == nil && resp.StatusCode == http.StatusOK {
				var apiResult struct {
					Result struct {
						Voices []struct {
							VoiceID    string `json:"voice_id"`
							CustomName string `json:"custom_name"`
							URI        string `json:"uri"`
						} `json:"voices"`
					} `json:"result"`
					Voices []struct {
						VoiceID    string `json:"voice_id"`
						CustomName string `json:"custom_name"`
						URI        string `json:"uri"`
					} `json:"voices"`
				}
				if json.Unmarshal(bodyBytes, &apiResult) == nil {
					customVoices := apiResult.Result.Voices
					if len(customVoices) == 0 {
						customVoices = apiResult.Voices
					}
					for _, v := range customVoices {
						id := v.URI
						if id == "" {
							id = v.VoiceID
						}
						name := v.CustomName
						if name == "" {
							name = v.VoiceID
						}
						voices = append(voices, voiceItem{ID: id, Name: name + " (自定义)"})
					}
				}
			}
		}
	}

	if voices == nil {
		voices = []voiceItem{}
	}

	c.JSON(http.StatusOK, gin.H{"data": voices})
}
