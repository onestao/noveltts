package handler

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"noveltts/internal/model"
	"noveltts/internal/provider"
)

type LegadoHandler struct {
	registry *provider.Registry
}

func NewLegadoHandler(reg *provider.Registry) *LegadoHandler {
	return &LegadoHandler{registry: reg}
}

func (h *LegadoHandler) TTS(c *gin.Context) {
	cfg := getAppConfig(c)

	if cfg.Server.AuthToken != "" {
		token := c.Query("token")
		if token == "" {
			if auth := c.GetHeader("Authorization"); strings.HasPrefix(auth, "Bearer ") {
				token = strings.TrimPrefix(auth, "Bearer ")
			} else if auth := c.GetHeader("Authorization"); auth != "" {
				token = auth
			}
		}
		if token != cfg.Server.AuthToken {
			log.Printf("[legado] auth failed, token mismatch")
			c.Header("Content-Type", "audio/mpeg")
			c.Status(http.StatusUnauthorized)
			return
		}
	}

	text := c.Query("text")
	if text == "" {
		text = c.Query("speakText")
	}
	if text == "" {
		c.Header("Content-Type", "audio/mpeg")
		c.Status(http.StatusOK)
		return
	}

	speedStr := c.Query("speed")
	var speed float64 = cfg.Defaults.Speed
	if speedStr != "" {
		if legadoRate, err := strconv.ParseFloat(speedStr, 64); err == nil {
			speed = mapLegadoSpeed(legadoRate)
		}
	}

	providerName := c.Query("provider")
	voice := c.Query("voice")
	modelName := c.Query("model")

	prov, err := h.resolveProvider(providerName, cfg)
	if err != nil {
		log.Printf("[legado] no provider: %v", err)
		c.Header("Content-Type", "audio/mpeg")
		c.Status(http.StatusOK)
		return
	}

	if voice == "" {
		for _, pc := range cfg.Providers {
			if pc.Name == cfg.Defaults.Provider {
				voice = pc.DefaultVoice
				break
			}
		}
	}

	if modelName == "" {
		for _, pc := range cfg.Providers {
			if pc.Name == cfg.Defaults.Provider {
				modelName = pc.DefaultModel
				break
			}
		}
	}

	ttsReq := &model.TTSRequest{
		Text:           text,
		Voice:          voice,
		Speed:          speed,
		ResponseFormat: "mp3",
		Model:          modelName,
	}

	for _, pc := range cfg.Providers {
		if pc.Name == cfg.Defaults.Provider && pc.Extra != nil {
			var extra struct {
				Style       string `json:"style"`
				Dialect     string `json:"dialect"`
				UserMessage string `json:"user_message"`
			}
			if json.Unmarshal(pc.Extra, &extra) == nil {
				ttsReq.Style = extra.Style
				ttsReq.Dialect = extra.Dialect
				ttsReq.UserMessage = extra.UserMessage
			}
			break
		}
	}

	body, contentType, err := prov.Synthesize(c.Request.Context(), ttsReq)
	if err != nil {
		log.Printf("[legado] synthesize error: %v", err)
		c.Header("Content-Type", "audio/mpeg")
		c.Status(http.StatusOK)
		return
	}
	defer body.Close()

	c.Header("Content-Type", contentType)
	c.Status(http.StatusOK)
	io.Copy(c.Writer, body)
}

func mapLegadoSpeed(legadoRate float64) float64 {
	if legadoRate < 0 {
		legadoRate = 0
	}
	if legadoRate > 15 {
		legadoRate = 15
	}
	return 0.25 + (legadoRate/15.0)*3.75
}

func (h *LegadoHandler) resolveProvider(name string, cfg *model.AppConfig) (provider.Provider, error) {
	if name != "" {
		if p, ok := h.registry.Get(name); ok {
			return p, nil
		}
	}
	return h.registry.GetDefault(cfg)
}
