package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"noveltts/internal/model"
	"noveltts/internal/provider"
)

type ModelsHandler struct {
	registry *provider.Registry
}

func NewModelsHandler(reg *provider.Registry) *ModelsHandler {
	return &ModelsHandler{registry: reg}
}

func (h *ModelsHandler) ListAll(c *gin.Context) {
	cfg := getAppConfig(c)
	var allModels []model.ModelInfo

	for _, pc := range cfg.Providers {
		prov, ok := h.registry.Get(pc.Name)
		if !ok {
			continue
		}
		models, err := prov.ListModels(c.Request.Context())
		if err != nil {
			continue
		}
		allModels = append(allModels, models...)
	}

	c.JSON(http.StatusOK, gin.H{"data": allModels})
}

func (h *ModelsHandler) ListByProvider(c *gin.Context) {
	providerName := c.Param("provider")
	prov, ok := h.registry.Get(providerName)
	if !ok {
		c.JSON(http.StatusNotFound, model.ErrorResponse{
			Error: model.ErrorDetail{Message: "provider not found", Type: "not_found_error"},
		})
		return
	}

	models, err := prov.ListModels(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error: model.ErrorDetail{Message: err.Error(), Type: "server_error"},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": models})
}

func (h *ModelsHandler) ListOpenAI(c *gin.Context) {
	cfg := getAppConfig(c)
	var allModels []model.ModelInfo

	for _, pc := range cfg.Providers {
		prov, ok := h.registry.Get(pc.Name)
		if !ok {
			continue
		}
		models, err := prov.ListModels(c.Request.Context())
		if err != nil {
			continue
		}
		allModels = append(allModels, models...)
	}

	type openAIModel struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		OwnedBy string `json:"owned_by"`
	}

	var result []openAIModel
	for _, m := range allModels {
		result = append(result, openAIModel{
			ID:      m.ID,
			Object:  "model",
			OwnedBy: m.Provider,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   result,
	})
}
