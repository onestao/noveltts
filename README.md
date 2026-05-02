# NovelTTS

Self-hosted TTS engine server for Legado (阅读) App and OpenAI-compatible API.

## Features

- **Legado Integration**: Direct HTTP TTS endpoint for Legado's read-aloud engine
- **OpenAI Compatible API**: `POST /v1/audio/speech` endpoint
- **Multi-Provider**: SiliconFlow, OpenAI, Xiaomi MiMo, MiniMax T2A
- **Web UI**: Alpine.js + Tailwind dashboard for configuration
- **Single Binary**: Go + Gin backend, ~15MB Docker image
- **Token Auth**: Optional access token for Legado and API endpoints
- **Config Persistence**: JSON file mounted via Docker Volume

## Quick Start

### Docker (Recommended)

```bash
docker run -d \
  --name novelts \
  -p 8080:8080 \
  -v ./data:/data \
  ghcr.io/onestao/noveltts:main
```

### docker-compose

```yaml
services:
  novelts:
    image: ghcr.io/onestao/noveltts:main
    ports:
      - "8080:8080"
    volumes:
      - ./data:/data
    restart: unless-stopped
```

### Build from Source

Requires Go 1.22+.

```bash
go mod tidy
go build -o novelts ./cmd/server
./noveltts
```

Open `http://localhost:8080` to configure providers.

## Legado Configuration

1. Open Legado App → Settings → Read Aloud Engine
2. Add custom HTTP TTS
3. URL: `http://<server-ip>:8080/legado/tts?text={{speakText}}&speed={{speakSpeed}}`
4. If auth is enabled, add header: `Authorization: Bearer <your-token>`

### Legado Header Setup

In Legado's HTTP TTS configuration, you can set custom headers. Add:

```
Authorization: Bearer your-token-here
```

Or pass the token in the URL:

```
http://<server-ip>:8080/legado/tts?text={{speakText}}&speed={{speakSpeed}}&token=your-token-here
```

## OpenAI Compatible API

```bash
# Synthesize speech
curl -X POST http://localhost:8080/v1/audio/speech \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-token" \
  -d '{"model":"tts-1","input":"你好世界","voice":"alloy","speed":1.0}' \
  --output speech.mp3

# List models
curl http://localhost:8080/v1/models
```

## Supported Providers

| Provider | Type | Base URL |
|----------|------|----------|
| SiliconFlow | `openai_compatible` | `https://api.siliconflow.cn/v1` |
| OpenAI | `openai_compatible` | `https://api.openai.com/v1` |
| Xiaomi MiMo | `mimo` | `https://api.xiaomimimo.com/v1` |
| MiniMax | `minimax` | `https://api.minimaxi.com/v1` |

## Configuration

Edit `data/config.json` or use the Web UI:

```json
{
  "server": { "port": 8080, "log_level": "info", "auth_token": "" },
  "providers": [
    {
      "name": "siliconflow",
      "type": "openai_compatible",
      "base_url": "https://api.siliconflow.cn/v1",
      "api_key": "sk-xxx",
      "default_model": "FunAudioLLM/CosyVoice2-0.5B",
      "default_voice": "FunAudioLLM/CosyVoice2-0.5B:alex"
    }
  ],
  "defaults": { "provider": "siliconflow", "speed": 1.0, "format": "mp3" }
}
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `NOVELTTS_CONFIG` | `data/config.json` | Config file path |
