package handler

import (
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"noveltts/internal/config"
	"noveltts/internal/model"
	"noveltts/internal/provider"
)

type OpenAIHandler struct {
	registry *provider.Registry
}

func NewOpenAIHandler(reg *provider.Registry) *OpenAIHandler {
	return &OpenAIHandler{registry: reg}
}

func (h *OpenAIHandler) Speech(c *gin.Context) {
	cfg := config.Get()
	if cfg.Server.AuthToken != "" {
		auth := c.GetHeader("Authorization")
		token := strings.TrimPrefix(auth, "Bearer ")
		if token != cfg.Server.AuthToken {
			c.JSON(http.StatusUnauthorized, model.ErrorResponse{
				Error: model.ErrorDetail{Message: "invalid or missing auth token", Type: "authentication_error"},
			})
			return
		}
	}

	var req model.OpenAIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.ErrorDetail{Message: "invalid request body", Type: "invalid_request_error"},
		})
		return
	}

	if req.Input == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.ErrorDetail{Message: "input is required", Type: "invalid_request_error"},
		})
		return
	}

	cfg = getAppConfig(c)
	prov, err := h.resolveProvider(req.Model, cfg)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: model.ErrorDetail{Message: err.Error(), Type: "invalid_request_error"},
		})
		return
	}

	speed := req.Speed
	if speed <= 0 {
		speed = cfg.Defaults.Speed
	}
	if speed <= 0 {
		speed = 1.0
	}

	format := req.ResponseFormat
	if format == "" {
		format = cfg.Defaults.Format
	}
	if format == "" {
		format = "mp3"
	}

	ttsReq := &model.TTSRequest{
		Text:           req.Input,
		Voice:          req.Voice,
		Speed:          speed,
		ResponseFormat: format,
		Model:          req.Model,
	}

	body, contentType, err := prov.Synthesize(c.Request.Context(), ttsReq)
	if err != nil {
		log.Printf("[openai] synthesize error: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error: model.ErrorDetail{Message: err.Error(), Type: "server_error"},
		})
		return
	}
	defer body.Close()

	c.Header("Content-Type", contentType)
	c.Status(http.StatusOK)
	io.Copy(c.Writer, body)
}

func (h *OpenAIHandler) resolveProvider(modelName string, cfg *model.AppConfig) (provider.Provider, error) {
	if modelName != "" {
		for _, pc := range cfg.Providers {
			if pc.DefaultModel == modelName {
				if p, ok := h.registry.Get(pc.Name); ok {
					return p, nil
				}
			}
		}
	}
	return h.registry.GetDefault(cfg)
}
