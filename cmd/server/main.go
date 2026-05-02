package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"

	noveltts "noveltts"
	"noveltts/internal/config"
	"noveltts/internal/handler"
	"noveltts/internal/middleware"
	"noveltts/internal/model"
	"noveltts/internal/provider"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	registry := provider.NewRegistry()
	if err := registry.LoadFromConfig(cfg); err != nil {
		log.Printf("warning: load providers: %v", err)
	}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGHUP)
		for range sigCh {
			log.Println("[main] reloading config...")
			newCfg, err := config.Reload()
			if err != nil {
				log.Printf("[main] reload error: %v", err)
				continue
			}
			if err := registry.LoadFromConfig(newCfg); err != nil {
				log.Printf("[main] reload providers error: %v", err)
			}
			log.Println("[main] config reloaded")
		}
	}()

	r := gin.Default()
	r.Use(middleware.CORS())
	r.Use(middleware.Logger())
	r.Use(injectConfig(cfg))

	webSub, err := fs.Sub(noveltts.WebFS, "web")
	if err != nil {
		log.Fatalf("embed web: %v", err)
	}

	openaiH := handler.NewOpenAIHandler(registry)
	legadoH := handler.NewLegadoHandler(registry)
	modelsH := handler.NewModelsHandler(registry)
	voicesH := handler.NewVoicesHandler(registry)
	configH := handler.NewConfigHandler(registry)

	r.POST("/v1/audio/speech", openaiH.Speech)
	r.GET("/v1/models", modelsH.ListOpenAI)
	r.GET("/legado/tts", legadoH.TTS)

	api := r.Group("/api")
	{
		api.GET("/models", modelsH.ListAll)
		api.GET("/models/:provider", modelsH.ListByProvider)
		api.GET("/voices", voicesH.ListAll)
		api.GET("/voices/:provider/:model", voicesH.ListByProviderModel)
		api.POST("/voices/preview", configH.PreviewVoice)

		api.GET("/config", configH.GetConfig)
		api.PUT("/config", configH.UpdateConfig)
		api.POST("/config/providers", configH.AddProvider)
		api.DELETE("/config/providers/:name", configH.DeleteProvider)
		api.PUT("/config/defaults", configH.UpdateDefaults)
		api.POST("/config/fetch-models", configH.FetchModels)
		api.POST("/config/fetch-voices", configH.FetchVoices)
	}

	handler.RegisterWebUI(r, webSub)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("[main] NovelTTS starting on %s", addr)
	log.Printf("[main] Legado URL: http://<host>%s/legado/tts?text={speakText}&speed={speakSpeed}", addr)
	log.Printf("[main] OpenAI API: http://<host>%s/v1/audio/speech", addr)
	log.Printf("[main] Web UI: http://<host>%s/", addr)

	if err := r.Run(addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func injectConfig(cfg *model.AppConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("appConfig", cfg)
		c.Next()
	}
}
