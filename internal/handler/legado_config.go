package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"noveltts/internal/config"
	"noveltts/internal/provider"
)

type LegadoConfigHandler struct {
	registry *provider.Registry
}

func NewLegadoConfigHandler(reg *provider.Registry) *LegadoConfigHandler {
	return &LegadoConfigHandler{registry: reg}
}

type legadoTTSConfig struct {
	ID               int64  `json:"id"`
	Name             string `json:"name"`
	URL              string `json:"url"`
	ContentType      string `json:"contentType"`
	ConcurrentRate   string `json:"concurrentRate"`
	EnabledCookieJar bool   `json:"enabledCookieJar"`
	Header           string `json:"header"`
	LoginUrl         string `json:"loginUrl"`
	LoginCheckJs     string `json:"loginCheckJs"`
	LastUpdateTime   int64  `json:"lastUpdateTime"`
}

func (h *LegadoConfigHandler) getBaseURL(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	return fmt.Sprintf("%s://%s", scheme, c.Request.Host)
}

func (h *LegadoConfigHandler) buildTTSURL(baseURL, providerName, voice, authToken string) string {
	url := fmt.Sprintf("%s/legado/tts?text={{java.encodeURI(speakText)}}&speed={{speakSpeed}}", baseURL)
	if providerName != "" {
		url += "&provider=" + providerName
	}
	if voice != "" {
		url += "&voice=" + voice
	}
	if authToken != "" {
		url += "&token=" + authToken
	}
	return url
}

func (h *LegadoConfigHandler) ListConfigs(c *gin.Context) {
	cfg := config.Get()
	baseURL := h.getBaseURL(c)
	authToken := cfg.Server.AuthToken

	var configs []legadoTTSConfig

	for _, pc := range cfg.Providers {
		prov, ok := h.registry.Get(pc.Name)
		if !ok {
			continue
		}

		voices, err := prov.ListVoices(c.Request.Context(), pc.DefaultModel)
		if err != nil {
			continue
		}

		if len(voices) == 0 {
			configs = append(configs, legadoTTSConfig{
				ID:               time.Now().UnixMilli(),
				Name:             pc.Name,
				URL:              h.buildTTSURL(baseURL, pc.Name, pc.DefaultVoice, authToken),
				ContentType:      "audio/mpeg",
				ConcurrentRate:   "5",
				EnabledCookieJar: false,
				Header:           "",
				LoginUrl:         "",
				LoginCheckJs:     "",
				LastUpdateTime:   time.Now().UnixMilli(),
			})
			continue
		}

		for _, v := range voices {
			voiceID := v.ID
			voiceName := v.Name
			if voiceName == "" {
				voiceName = voiceID
			}
			configs = append(configs, legadoTTSConfig{
				ID:               time.Now().UnixMilli() + int64(len(configs)),
				Name:             fmt.Sprintf("%s - %s", pc.Name, voiceName),
				URL:              h.buildTTSURL(baseURL, pc.Name, voiceID, authToken),
				ContentType:      "audio/mpeg",
				ConcurrentRate:   "5",
				EnabledCookieJar: false,
				Header:           "",
				LoginUrl:         "",
				LoginCheckJs:     "",
				LastUpdateTime:   time.Now().UnixMilli(),
			})
		}
	}

	if configs == nil {
		configs = []legadoTTSConfig{}
	}

	c.JSON(http.StatusOK, configs)
}

func (h *LegadoConfigHandler) GetSingleConfig(c *gin.Context) {
	providerName := c.Query("provider")
	voiceID := c.Query("voice")
	voiceName := c.Query("name")

	cfg := config.Get()
	baseURL := h.getBaseURL(c)
	authToken := cfg.Server.AuthToken

	name := voiceName
	if name == "" {
		name = providerName
	}
	if voiceID != "" && voiceName == "" {
		name = fmt.Sprintf("%s - %s", providerName, voiceID)
	}

	oneCfg := legadoTTSConfig{
		ID:               time.Now().UnixMilli(),
		Name:             name,
		URL:              h.buildTTSURL(baseURL, providerName, voiceID, authToken),
		ContentType:      "audio/mpeg",
		ConcurrentRate:   "5",
		EnabledCookieJar: false,
		Header:           "",
		LoginUrl:         "",
		LoginCheckJs:     "",
		LastUpdateTime:   time.Now().UnixMilli(),
	}

	c.JSON(http.StatusOK, oneCfg)
}

func (h *LegadoConfigHandler) ImportGuide(c *gin.Context) {
	cfg := config.Get()
	baseURL := h.getBaseURL(c)
	hasAuth := cfg.Server.AuthToken != ""

	authWarning := ""
	if hasAuth {
		authWarning = fmt.Sprintf(`<div class="warn">⚠️ 已启用认证。在阅读 App 中还需配置 Header：<br><code>Authorization: Bearer %s</code></div>`, cfg.Server.AuthToken)
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Legado 导入指南</title>
<style>
body{font-family:-apple-system,sans-serif;max-width:600px;margin:40px auto;padding:0 20px;color:#333}
h1{font-size:20px;margin-bottom:20px}
.step{background:#f5f5f5;border-radius:8px;padding:16px;margin:12px 0}
.step h3{margin:0 0 8px;font-size:14px;color:#666}
.step p{margin:0;font-size:14px;line-height:1.6}
code{background:#e8e8e8;padding:2px 6px;border-radius:4px;font-size:13px;word-break:break-all}
.url-box{background:#fff;border:1px solid #ddd;border-radius:8px;padding:12px;margin:16px 0;font-family:monospace;font-size:12px;word-break:break-all;cursor:pointer}
.url-box:hover{border-color:#4a90d9}
.btn{display:inline-block;background:#4a90d9;color:#fff;padding:10px 20px;border-radius:8px;text-decoration:none;font-size:14px;margin:8px 4px 8px 0;border:none;cursor:pointer}
.btn:hover{background:#357abd}
.btn-green{background:#34c759}.btn-green:hover{background:#2da44e}
.warn{background:#fff3cd;border:1px solid #ffc107;border-radius:8px;padding:12px;margin:16px 0;font-size:13px}
</style></head><body>
<h1>📖 导入到阅读 (Legado)</h1>
<div class="step"><h3>方式一：一键导入（推荐）</h3>
<p>点击下方按钮，如果手机已安装阅读 App 会自动弹出导入页面。</p>
<button class="btn btn-green" onclick="importAll()">🚀 一键导入全部音色</button>
<button class="btn" onclick="importSingle()">📥 导入单个音色</button></div>
<div class="step"><h3>方式二：手动配置</h3>
<p>1. 打开阅读 App → 设置 → 朗读引擎</p>
<p>2. 添加自定义 HTTP TTS</p>
<p>3. URL 填写：</p><div class="url-box" onclick="copyUrl(this)" id="manualUrl">%s/legado/tts?text={%s{speakText}}&speed={%s{speakSpeed}}</div>
<p>4. Content-Type 填写：<code>audio/mpeg</code></p>
</div>
%s
<script>
var baseUrl = '%s';
var importUrl = baseUrl + '/api/legado/tts-configs';
function importAll(){
  var url = 'legado://import/httpTTS?src=' + encodeURIComponent(importUrl);
  window.location.href = url;
  setTimeout(function(){alert('如果阅读 App 未打开，请使用方式二手动配置')},2000);
}
function importSingle(){
  var voice = prompt('请输入音色 ID（留空使用默认）：','');
  var url = baseUrl + '/api/legado/tts-config?provider=&voice=' + encodeURIComponent(voice||'');
  var legadoUrl = 'legado://import/httpTTS?src=' + encodeURIComponent(url);
  window.location.href = legadoUrl;
  setTimeout(function(){alert('如果阅读 App 未打开，请使用方式二手动配置')},2000);
}
function copyUrl(el){
  navigator.clipboard.writeText(el.textContent).then(function(){alert('已复制')});
}
</script></body></html>`, baseURL, "{{java.encode", "URI(speak", "}}", authWarning, baseURL)

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}
