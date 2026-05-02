package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"noveltts/internal/model"
	"noveltts/internal/provider"
)

type VoicesHandler struct {
	registry *provider.Registry
}

func NewVoicesHandler(reg *provider.Registry) *VoicesHandler {
	return &VoicesHandler{registry: reg}
}

func (h *VoicesHandler) ListAll(c *gin.Context) {
	cfg := getAppConfig(c)
	type voiceWithProvider struct {
		model.VoiceInfo
		Provider string `json:"provider"`
	}

	var allVoices []voiceWithProvider

	for _, pc := range cfg.Providers {
		prov, ok := h.registry.Get(pc.Name)
		if !ok {
			continue
		}
		voices, err := prov.ListVoices(c.Request.Context(), pc.DefaultModel)
		if err != nil {
			continue
		}
		for _, v := range voices {
			allVoices = append(allVoices, voiceWithProvider{
				VoiceInfo: v,
				Provider:  pc.Name,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": allVoices})
}

func (h *VoicesHandler) ListByProviderModel(c *gin.Context) {
	providerName := c.Param("provider")
	modelID := c.Param("model")

	prov, ok := h.registry.Get(providerName)
	if !ok {
		c.JSON(http.StatusNotFound, model.ErrorResponse{
			Error: model.ErrorDetail{Message: "provider not found", Type: "not_found_error"},
		})
		return
	}

	voices, err := prov.ListVoices(c.Request.Context(), modelID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error: model.ErrorDetail{Message: err.Error(), Type: "server_error"},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": voices})
}
