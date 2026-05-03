package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"noveltts/internal/model"
)

type doubaoExtra struct {
	Cookies []string `json:"cookies"`
	WsURL   string   `json:"ws_url,omitempty"`
}

type DoubaoProvider struct {
	name      string
	cookies   []string
	wsURL     string
	mu        sync.RWMutex
	cookieIdx int
}

func NewDoubaoProvider(cfg model.ProviderConfig) *DoubaoProvider {
	var cookies []string
	var wsURL string
	if cfg.Extra != nil {
		var extra doubaoExtra
		if json.Unmarshal(cfg.Extra, &extra) == nil {
			cookies = extra.Cookies
			wsURL = extra.WsURL
		}
	}
	if wsURL == "" {
		wsURL = "wss://amantha.doubao.com/latasr/api/ws/tts/v1"
	}
	return &DoubaoProvider{
		name:    cfg.Name,
		cookies: cookies,
		wsURL:   wsURL,
	}
}

func (p *DoubaoProvider) Name() string { return p.name }
func (p *DoubaoProvider) Type() string { return "doubao" }

type doubaoTTSRequest struct {
	AID       int    `json:"aid"`
	Speaker   string `json:"speaker"`
	SpeechRate int   `json:"speech_rate"`
	Pitch     int    `json:"pitch"`
	Language  string `json:"language"`
	PkgType   string `json:"pkg_type"`
	SysRegion string `json:"sys_region"`
	UseOlympus int  `json:"use_olympus_account"`
	CookieID  string `json:"cookieId,omitempty"`
	DeviceID  string `json:"deviceId,omitempty"`
	TeaUUID   string `json:"tea_uuid,omitempty"`
	WebID     string `json:"web_id,omitempty"`
}

type doubaoWSMessage struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data,omitempty"`
}

type doubaoAudioData struct {
	Audio  string `json:"audio"`
	Status int    `json:"status"`
}

func (p *DoubaoProvider) Synthesize(ctx context.Context, req *model.TTSRequest) (io.ReadCloser, string, error) {
	cookie := p.nextCookie()
	if cookie == "" {
		log.Printf("[doubao] ERROR: no cookie configured")
		return nil, "", fmt.Errorf("no cookie configured for Doubao provider")
	}
	log.Printf("[doubao] cookie len=%d, first20=%s", len(cookie), truncate(cookie, 20))

	speaker := req.Voice
	if speaker == "" {
		speaker = "zh_female_wenroutaozi_uranus_bigtts"
	}

	speed := int(req.Speed * 5)
	if speed == 0 {
		speed = 5
	}

	wsURL := p.wsURL
	log.Printf("[doubao] ws_url=%s speaker=%s speed=%d text_len=%d", wsURL, speaker, speed, len(req.Text))

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	headers := http.Header{}
	headers.Set("Cookie", cookie)
	headers.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	headers.Set("Origin", "https://www.doubao.com")
	headers.Set("Accept-Language", "zh,zh-CN;q=0.9")

	log.Printf("[doubao] dialing websocket...")
	conn, resp, err := dialer.DialContext(ctx, wsURL, headers)
	if err != nil {
		log.Printf("[doubao] ERROR: websocket dial failed: %v", err)
		if resp != nil {
			log.Printf("[doubao] handshake response: status=%d", resp.StatusCode)
		}
		return nil, "", fmt.Errorf("websocket dial: %w", err)
	}
	defer conn.Close()
	log.Printf("[doubao] websocket connected, resp_status=%d", resp.StatusCode)

	ttsReq := doubaoTTSRequest{
		AID:        497858,
		Speaker:    speaker,
		SpeechRate: speed,
		Pitch:      5,
		Language:   "zh",
		PkgType:    "release_version",
		SysRegion:  "CN",
		UseOlympus: 1,
	}

	reqJSON, _ := json.Marshal(ttsReq)
	log.Printf("[doubao] sending tts request: %s", string(reqJSON))
	if err := conn.WriteMessage(websocket.TextMessage, reqJSON); err != nil {
		log.Printf("[doubao] ERROR: send tts request: %v", err)
		return nil, "", fmt.Errorf("send tts request: %w", err)
	}

	textBytes := []byte(req.Text)
	log.Printf("[doubao] sending text: len=%d, first50=%s", len(req.Text), truncate(req.Text, 50))
	if err := conn.WriteMessage(websocket.BinaryMessage, textBytes); err != nil {
		log.Printf("[doubao] ERROR: send text: %v", err)
		return nil, "", fmt.Errorf("send text: %w", err)
	}

	var audioBuf bytes.Buffer
	done := false
	msgCount := 0

	log.Printf("[doubao] waiting for messages...")
	for !done {
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			if ctx.Err() != nil {
				log.Printf("[doubao] ERROR: context cancelled")
				return nil, "", ctx.Err()
			}
			log.Printf("[doubao] ERROR: read message: %v (msgCount=%d)", err, msgCount)
			break
		}

		msgCount++
		if msgType == websocket.TextMessage {
			var wsMsg doubaoWSMessage
			if json.Unmarshal(msg, &wsMsg) != nil {
				log.Printf("[doubao] text msg (unmarshal failed): %s", truncate(string(msg), 200))
				continue
			}

			log.Printf("[doubao] event=%s", wsMsg.Event)
			switch wsMsg.Event {
			case "finish":
				done = true
				log.Printf("[doubao] finished, audio_buf=%d bytes", audioBuf.Len())
			case "error":
				var errMsg struct {
					Message string `json:"message"`
				}
				json.Unmarshal(wsMsg.Data, &errMsg)
				log.Printf("[doubao] ERROR: server error: %s", errMsg.Message)
				return nil, "", fmt.Errorf("doubao error: %s", errMsg.Message)
			case "audio":
				var audioData doubaoAudioData
				if json.Unmarshal(wsMsg.Data, &audioData) == nil && audioData.Audio != "" {
					decoded, err := decodeDoubaoAudio(audioData.Audio)
					if err == nil {
						audioBuf.Write(decoded)
						log.Printf("[doubao] audio chunk: decoded=%d bytes, total=%d", len(decoded), audioBuf.Len())
					} else {
						log.Printf("[doubao] audio decode error: %v", err)
					}
				}
			}
		} else if msgType == websocket.BinaryMessage {
			audioBuf.Write(msg)
			log.Printf("[doubao] binary msg: %d bytes, total=%d", len(msg), audioBuf.Len())
		}
	}

	if audioBuf.Len() == 0 {
		return nil, "", fmt.Errorf("no audio data received from Doubao")
	}

	log.Printf("[doubao] synthesized %d bytes", audioBuf.Len())
	return io.NopCloser(bytes.NewReader(audioBuf.Bytes())), "audio/mpeg", nil
}

func decodeDoubaoAudio(data string) ([]byte, error) {
	if len(data) > 4 && (data[:4] == "SUQz" || data[:2] == "//") {
		decoded := make([]byte, len(data))
		n, err := base64Decode(data, decoded)
		if err != nil {
			return nil, err
		}
		return decoded[:n], nil
	}

	raw := []byte(data)
	result := make([]byte, 0, len(raw))
	for _, b := range raw {
		if b >= '0' && b <= '9' {
			result = append(result, b-'0')
		} else if b >= 'a' && b <= 'f' {
			result = append(result, b-'a'+10)
		} else if b >= 'A' && b <= 'F' {
			result = append(result, b-'A'+10)
		}
	}

	if len(result) > 0 && len(result)%2 == 0 {
		decoded := make([]byte, len(result)/2)
		for i := 0; i < len(result); i += 2 {
			decoded[i/2] = (result[i] << 4) | result[i+1]
		}
		return decoded, nil
	}

	return []byte(data), nil
}

func base64Decode(s string, dst []byte) (int, error) {
	const base64Chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	val := 0
	bits := 0
	n := 0
	for _, c := range s {
		idx := strings.IndexRune(base64Chars, c)
		if idx < 0 {
			if c == '=' {
				continue
			}
			continue
		}
		val = (val << 6) | idx
		bits += 6
		if bits >= 8 {
			bits -= 8
			dst[n] = byte(val >> bits)
			n++
			val &= (1 << bits) - 1
		}
	}
	return n, nil
}

func (p *DoubaoProvider) nextCookie() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.cookies) == 0 {
		return ""
	}
	cookie := p.cookies[p.cookieIdx%len(p.cookies)]
	p.cookieIdx++
	return cookie
}

func (p *DoubaoProvider) ListModels(ctx context.Context) ([]model.ModelInfo, error) {
	return []model.ModelInfo{
		{ID: "doubao-tts", Name: "豆包 TTS (WebSocket)", Provider: p.name},
	}, nil
}

func (p *DoubaoProvider) ListVoices(ctx context.Context, modelID string) ([]model.VoiceInfo, error) {
	return []model.VoiceInfo{
		{ID: "zh_female_wenroutaozi_uranus_bigtts", Name: "温柔桃子", Gender: "female"},
		{ID: "zh_male_nuanxinshizhe_mars_bigtts", Name: "磁性俊宇", Gender: "male"},
		{ID: "zh_female_xiaohe_conversation_wvae_bigtts", Name: "阳光甜妹", Gender: "female"},
		{ID: "zh_female_wenroutaozi_v2_mars_bigtts", Name: "温柔桃子(经典)", Gender: "female"},
		{ID: "zh_female_f261_conversation_wvae_bigtts", Name: "邻家女孩", Gender: "female"},
		{ID: "zh_female_sophie_conversation_wvae_bigtts", Name: "魅力苏菲", Gender: "female"},
		{ID: "zh_female_yuanqinvyou_wvae_bigtts", Name: "撒娇学妹", Gender: "female"},
		{ID: "zh_male_linjiananhai_moon_bigtts", Name: "邻家男孩", Gender: "male"},
		{ID: "zh_male_M100_conversation_wvae_bigtts", Name: "悠悠君子", Gender: "male"},
		{ID: "zh_male_ahu_conversation_wvae_bigtts", Name: "温暖阿虎", Gender: "male"},
		{ID: "zh_male_m286_conversation_wvae_bigtts", Name: "少年梓辛", Gender: "male"},
		{ID: "zh_male_qingyiyuxuan_mars_bigtts", Name: "阳光阿辰", Gender: "male"},
		{ID: "ICL_c021bc19bf92", Name: "腹黑霸总", Gender: "male"},
		{ID: "ICL_e0b9b93ee322", Name: "冷酷霸总", Gender: "male"},
		{ID: "zh_male_aojiaobazong_wvae_bigtts", Name: "傲娇霸总", Gender: "male"},
		{ID: "ICL_d4d40acd33dd", Name: "霸道总裁", Gender: "male"},
		{ID: "zh_male_cheng_mars_bigtts", Name: "温柔子言", Gender: "male"},
		{ID: "zh_male_litiebanzi_mars_bigtts", Name: "率性阿哲", Gender: "male"},
		{ID: "ICL_df4fc4d1ce4b", Name: "温柔陆辰", Gender: "male"},
		{ID: "zh_male_shenyeboke_wvae_bigtts", Name: "深夜播客", Gender: "male"},
		{ID: "ICL_6acf86286e24", Name: "甜美小雪", Gender: "female"},
		{ID: "ICL_16cd9a58768e", Name: "清冷阿梦", Gender: "female"},
		{ID: "zh_male_dongfanghaoran_moon_bigtts", Name: "东方浩然", Gender: "male"},
		{ID: "ICL_72afa6c5dc07", Name: "病娇少爷", Gender: "male"},
		{ID: "zh_male_junlangxize_mars_bigtts", Name: "清爽男大", Gender: "male"},
		{ID: "ICL_9b3bc6941076", Name: "清朗宇澄", Gender: "male"},
		{ID: "ICL_932b3f52bf3d", Name: "奶音俊少", Gender: "male"},
		{ID: "ICL_5a413fbc14fc", Name: "沉稳皓轩", Gender: "male"},
		{ID: "ICL_0ce6ef379e73", Name: "温柔俊彦", Gender: "male"},
	}, nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
