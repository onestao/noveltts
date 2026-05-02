const API = {
  async get(url) {
    const resp = await fetch(url);
    if (!resp.ok) {
      const err = await resp.json().catch(() => ({ error: { message: resp.statusText } }));
      throw new Error(err.error?.message || resp.statusText);
    }
    return resp.json();
  },

  async post(url, body) {
    const resp = await fetch(url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    if (!resp.ok) {
      const err = await resp.json().catch(() => ({ error: { message: resp.statusText } }));
      throw new Error(err.error?.message || resp.statusText);
    }
    return resp.json();
  },

  async postBinary(url, body) {
    const resp = await fetch(url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    if (!resp.ok) {
      const err = await resp.json().catch(() => ({ error: { message: resp.statusText } }));
      throw new Error(err.error?.message || resp.statusText);
    }
    return resp.blob();
  },

  async put(url, body) {
    const resp = await fetch(url, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    if (!resp.ok) {
      const err = await resp.json().catch(() => ({ error: { message: resp.statusText } }));
      throw new Error(err.error?.message || resp.statusText);
    }
    return resp.json();
  },

  async del(url) {
    const resp = await fetch(url, { method: 'DELETE' });
    if (!resp.ok) {
      const err = await resp.json().catch(() => ({ error: { message: resp.statusText } }));
      throw new Error(err.error?.message || resp.statusText);
    }
    return resp.json();
  },

  getConfig() { return this.get('/api/config'); },
  updateConfig(cfg) { return this.put('/api/config', cfg); },
  addProvider(p) { return this.post('/api/config/providers', p); },
  deleteProvider(name) { return this.del('/api/config/providers/' + encodeURIComponent(name)); },
  updateDefaults(d) { return this.put('/api/config/defaults', d); },
  getVoices() { return this.get('/api/voices'); },
  fetchModels(base_url, api_key) { return this.post('/api/config/fetch-models', { base_url, api_key }); },
  fetchVoices(base_url, api_key) { return this.post('/api/config/fetch-voices', { base_url, api_key }); },
  previewVoice(provider, voice, model, text) { return this.postBinary('/api/voices/preview', { provider, voice, model, text }); },
};
