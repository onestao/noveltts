function app() {
  return {
    page: 'dashboard',
    sidebarOpen: false,
    config: {},
    voices: [],
    showAddProvider: false,
    providerForm: { name: '', type: 'openai_compatible', base_url: '', api_key: '', default_model: '', default_voice: '', _editing: false },
    defaultsForm: { provider: '', speed: 1.0, format: 'mp3' },
    authToken: '',
    toast: { show: false, message: '', type: 'success' },
    availableModels: [],
    availableVoices: [],
    fetchingModels: false,
    fetchingVoices: false,

    // Voices page
    previewText: '这是一段测试语音，用于试听音色效果。今天天气真不错，适合出去走走。',
    previewingVoice: null,
    previewAudio: null,
    selectedVoices: {},

    // Legado page
    legadoConfigs: [],

    get baseUrl() {
      return window.location.origin;
    },

    get legadoUrl() {
      let url = this.baseUrl + '/legado/tts?text={speakText}&speed={speakSpeed}';
      if (this.config.server?.auth_token) {
        url += '&token=' + this.config.server.auth_token;
      }
      return url;
    },

    async init() {
      await this.loadConfig();
      await this.loadVoices();
      await this.loadLegadoConfigs();
    },

    async loadConfig() {
      try {
        this.config = await API.getConfig();
        this.defaultsForm = {
          provider: this.config.defaults?.provider || '',
          speed: this.config.defaults?.speed || 1.0,
          format: this.config.defaults?.format || 'mp3',
        };
        this.authToken = '';
      } catch (e) {
        this.showToast('加载配置失败：' + e.message, 'error');
      }
    },

    async loadVoices() {
      try {
        const resp = await API.getVoices();
        this.voices = resp.data || [];
      } catch (e) {
        this.voices = [];
      }
    },

    async previewVoice(provider, voiceId, modelId) {
      if (this.previewingVoice) return;
      this.previewingVoice = voiceId;
      try {
        const blob = await API.previewVoice(provider, voiceId, modelId, this.previewText);
        if (this.previewAudio) {
          this.previewAudio.pause();
          URL.revokeObjectURL(this.previewAudio.src);
        }
        const url = URL.createObjectURL(blob);
        this.previewAudio = new Audio(url);
        this.previewAudio.play();
        this.previewAudio.onended = () => { this.previewingVoice = null; };
        this.previewAudio.onerror = () => { this.previewingVoice = null; };
      } catch (e) {
        this.showToast('试听失败：' + e.message, 'error');
        this.previewingVoice = null;
      }
    },

    stopPreview() {
      if (this.previewAudio) {
        this.previewAudio.pause();
        this.previewAudio.currentTime = 0;
        URL.revokeObjectURL(this.previewAudio.src);
        this.previewAudio = null;
      }
      this.previewingVoice = null;
    },

    toggleVoiceSelect(provider, voiceId) {
      const key = provider + '::' + voiceId;
      if (this.selectedVoices[key]) {
        delete this.selectedVoices[key];
      } else {
        this.selectedVoices[key] = { provider, voice: voiceId };
      }
    },

    isVoiceSelected(provider, voiceId) {
      return !!this.selectedVoices[provider + '::' + voiceId];
    },

    get selectedVoiceCount() {
      return Object.keys(this.selectedVoices).length;
    },

    resetProviderForm() {
      this.providerForm = { name: '', type: 'openai_compatible', base_url: '', api_key: '', cookie: '', default_model: '', default_voice: '', mimo_style: '', mimo_dialect: '', mimo_user_message: '', _editing: false };
      this.availableModels = [];
      this.availableVoices = [];
    },

    onProviderTypeChange() {
      this.availableModels = [];
      this.availableVoices = [];
      const t = this.providerForm.type;
      if (t === 'mimo') {
        this.providerForm.base_url = 'https://api.xiaomimimo.com/v1';
        this.providerForm.name = this.providerForm.name || 'mimo';
      } else if (t === 'doubao') {
        this.providerForm.base_url = 'wss://amantha.doubao.com';
        this.providerForm.name = this.providerForm.name || 'doubao';
        this.providerForm.default_model = 'doubao-tts';
      } else if (t === 'minimax') {
        this.providerForm.base_url = 'https://api.minimaxi.com/v1';
        this.providerForm.name = this.providerForm.name || 'minimax';
      } else {
        this.providerForm.base_url = '';
        this.providerForm.name = this.providerForm.name || '';
      }
    },

    editProvider(p) {
      let mimoStyle = '', mimoDialect = '', mimoUserMessage = '';
      if (p.extra) {
        try {
          const e = typeof p.extra === 'string' ? JSON.parse(p.extra) : p.extra;
          mimoStyle = e.style || '';
          mimoDialect = e.dialect || '';
          mimoUserMessage = e.user_message || '';
        } catch(e) {}
      }
      this.providerForm = {
        name: p.name,
        type: p.type,
        base_url: p.base_url,
        api_key: '',
        default_model: p.default_model || '',
        default_voice: p.default_voice || '',
        mimo_style: mimoStyle,
        mimo_dialect: mimoDialect,
        mimo_user_message: mimoUserMessage,
        _editing: true,
        _originalKey: p.api_key,
      };
      this.availableModels = [];
      this.availableVoices = [];
      this.showAddProvider = true;
    },

    async fetchModels() {
      if (!this.providerForm.base_url) {
        this.showToast('请先填写 API 地址', 'error');
        return;
      }
      this.fetchingModels = true;
      this.availableModels = [];
      try {
        const resp = await API.fetchModels(this.providerForm.base_url, this.providerForm.api_key);
        this.availableModels = resp.data || [];
        if (this.availableModels.length === 0) {
          this.showToast('未获取到模型', 'error');
        } else {
          this.showToast('获取到 ' + this.availableModels.length + ' 个模型');
        }
      } catch (e) {
        this.showToast('获取模型失败：' + e.message, 'error');
      } finally {
        this.fetchingModels = false;
      }
    },

    async fetchVoices() {
      if (!this.providerForm.base_url) {
        this.showToast('请先填写 API 地址', 'error');
        return;
      }
      this.fetchingVoices = true;
      this.availableVoices = [];
      try {
        const resp = await API.fetchVoices(this.providerForm.base_url, this.providerForm.api_key);
        const data = resp.data || resp.result || resp;
        if (Array.isArray(data)) {
          this.availableVoices = data;
        } else if (data && Array.isArray(data.voices)) {
          this.availableVoices = data.voices;
        } else {
          this.availableVoices = [];
        }
        if (this.availableVoices.length === 0) {
          this.showToast('未获取到音色', 'error');
        } else {
          this.showToast('获取到 ' + this.availableVoices.length + ' 个音色');
        }
      } catch (e) {
        this.showToast('获取音色失败：' + e.message, 'error');
      } finally {
        this.fetchingVoices = false;
      }
    },

    selectVoice(voiceId) {
      this.providerForm.default_voice = voiceId;
    },

    async saveProvider() {
      if (!this.providerForm.name) {
        this.showToast('请填写引擎名称', 'error');
        return;
      }
      if (!this.providerForm.base_url) {
        this.showToast('请填写 API 地址', 'error');
        return;
      }

      const payload = {
        name: this.providerForm.name,
        type: this.providerForm.type,
        base_url: this.providerForm.base_url,
        api_key: this.providerForm.api_key || this.providerForm._originalKey || '',
        default_model: this.providerForm.default_model,
        default_voice: this.providerForm.default_voice,
      };

      if (this.providerForm.type === 'doubao' && this.providerForm.cookie) {
        payload.extra = { cookies: this.providerForm.cookie.split(/[;\n]/).map(c => c.trim()).filter(c => c) };
      }

      if (this.providerForm.type === 'mimo') {
        const mimoExtra = {};
        if (this.providerForm.mimo_style) mimoExtra.style = this.providerForm.mimo_style;
        if (this.providerForm.mimo_dialect) mimoExtra.dialect = this.providerForm.mimo_dialect;
        if (this.providerForm.mimo_user_message) mimoExtra.user_message = this.providerForm.mimo_user_message;
        if (Object.keys(mimoExtra).length > 0) payload.extra = mimoExtra;
      }

      try {
        await API.addProvider(payload);
        this.showAddProvider = false;
        await this.loadConfig();
        await this.loadVoices();
        this.showToast(this.providerForm._editing ? '引擎已更新' : '引擎已添加');
      } catch (e) {
        this.showToast('保存失败：' + e.message, 'error');
      }
    },

    async deleteProvider(name) {
      if (!confirm('确定删除引擎「' + name + '」？')) return;
      try {
        await API.deleteProvider(name);
        await this.loadConfig();
        await this.loadVoices();
        this.showToast('引擎已删除');
      } catch (e) {
        this.showToast('删除失败：' + e.message, 'error');
      }
    },

    async saveDefaults() {
      try {
        await API.updateDefaults(this.defaultsForm);
        await this.loadConfig();
        this.showToast('默认设置已保存');
      } catch (e) {
        this.showToast('保存失败：' + e.message, 'error');
      }
    },

    async saveAuthToken() {
      try {
        const cfg = JSON.parse(JSON.stringify(this.config));
        cfg.server.auth_token = this.authToken;
        for (const p of cfg.providers) {
          p.api_key = '';
        }
        await API.updateConfig(cfg);
        await this.loadConfig();
        this.showToast('认证设置已保存');
      } catch (e) {
        this.showToast('保存失败：' + e.message, 'error');
      }
    },

    copyLegadoUrl() {
      navigator.clipboard.writeText(this.legadoUrl).then(() => {
        this.showToast('URL 已复制到剪贴板');
      }).catch(() => {
        this.showToast('复制失败，请手动复制', 'error');
      });
    },

    async loadLegadoConfigs() {
      try {
        const resp = await fetch('/api/legado/tts-configs');
        const data = await resp.json();
        this.legadoConfigs = Array.isArray(data) ? data : [];
      } catch (e) {
        this.legadoConfigs = [];
      }
    },

    importAllToLegado() {
      const url = this.baseUrl + '/api/legado/tts-configs';
      const legadoUrl = 'legado://import/httpTTS?src=' + encodeURIComponent(url);
      window.location.href = legadoUrl;
      setTimeout(() => {
        this.showToast('如果阅读 App 未打开，请使用「查看导入指南」手动配置', 'error');
      }, 2000);
    },

    importSingleToLegado(cfg) {
      const url = this.baseUrl + '/api/legado/tts-config?provider=' + encodeURIComponent(cfg.name.split(' - ')[0]) + '&voice=' + encodeURIComponent(cfg.name.split(' - ')[1] || '');
      const legadoUrl = 'legado://import/httpTTS?src=' + encodeURIComponent(url);
      window.location.href = legadoUrl;
      setTimeout(() => {
        this.showToast('如果阅读 App 未打开，请手动复制配置', 'error');
      }, 2000);
    },

    copyLegadoConfig(cfg) {
      const configJson = JSON.stringify(cfg, null, 2);
      navigator.clipboard.writeText(configJson).then(() => {
        this.showToast('配置已复制到剪贴板');
      }).catch(() => {
        this.showToast('复制失败', 'error');
      });
    },

    openLegadoGuide() {
      window.open('/legado/guide', '_blank');
    },

    showToast(message, type = 'success') {
      this.toast = { show: true, message, type };
      setTimeout(() => { this.toast.show = false; }, 3000);
    },
  };
}
