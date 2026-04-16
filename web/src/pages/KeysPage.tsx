import { FormEvent, useEffect, useState } from 'react';
import { api, type AdminKey, type CreateKeyPayload } from '../api';

const emptyForm: CreateKeyPayload = {
  name: '',
  provider: 'openai',
  base_url: '',
  api_key: '',
  enabled: true
};

export default function KeysPage(): JSX.Element {
  const [items, setItems] = useState<AdminKey[]>([]);
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [form, setForm] = useState<CreateKeyPayload>(emptyForm);

  const loadKeys = async (): Promise<void> => {
    setLoading(true);
    setError(null);
    try {
      const response = await api.getKeys();
      setItems(response.items);
    } catch (loadError) {
      const message = loadError instanceof Error ? loadError.message : 'Failed to load keys.';
      setError(message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void loadKeys();
  }, []);

  const handleSubmit = async (event: FormEvent<HTMLFormElement>): Promise<void> => {
    event.preventDefault();
    setSubmitting(true);
    setError(null);
    try {
      await api.createKey(form);
      setForm(emptyForm);
      await loadKeys();
    } catch (submitError) {
      const message = submitError instanceof Error ? submitError.message : 'Failed to create key.';
      setError(message);
    } finally {
      setSubmitting(false);
    }
  };

  const handleProbe = async (id: number): Promise<void> => {
    setError(null);
    try {
      await api.probeKey(id);
      await loadKeys();
    } catch (probeError) {
      const message = probeError instanceof Error ? probeError.message : 'Failed to probe key.';
      setError(message);
    }
  };

  const handleDelete = async (id: number): Promise<void> => {
    setError(null);
    try {
      await api.deleteKey(id);
      await loadKeys();
    } catch (deleteError) {
      const message = deleteError instanceof Error ? deleteError.message : 'Failed to delete key.';
      setError(message);
    }
  };

  return (
    <section className="page-section">
      <header className="page-header">
        <h2>Keys</h2>
      </header>

      <form className="card form-grid" onSubmit={handleSubmit}>
        <h3>Add upstream key</h3>
        <label>
          Label
          <input value={form.name} onChange={(event) => setForm((current) => ({ ...current, name: event.target.value }))} required />
        </label>
        <label>
          Provider
          <select value={form.provider} onChange={(event) => setForm((current) => ({ ...current, provider: event.target.value }))}>
            <option value="openai">OpenAI</option>
            <option value="anthropic">Anthropic</option>
            <option value="gemini">Gemini</option>
          </select>
        </label>
        <label>
          Base URL
          <input value={form.base_url} onChange={(event) => setForm((current) => ({ ...current, base_url: event.target.value }))} placeholder="optional" />
        </label>
        <label className="full-width">
          API Key
          <input value={form.api_key} onChange={(event) => setForm((current) => ({ ...current, api_key: event.target.value }))} required />
        </label>
        <button type="submit" disabled={submitting}>{submitting ? 'Saving…' : 'Save key'}</button>
      </form>

      {loading ? <p className="muted">Loading keys…</p> : null}
      {error ? <p className="error-text">{error}</p> : null}

      <div className="table-card">
        <table>
          <thead>
            <tr>
              <th>Label</th>
              <th>Provider</th>
              <th>Status</th>
              <th>Enabled</th>
              <th>Last Used</th>
              <th>Last Error</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {items.length > 0 ? (
              items.map((item) => (
                <tr key={item.id}>
                  <td>{item.label}</td>
                  <td>{item.provider}</td>
                  <td>{item.status}</td>
                  <td>{item.enabled ? 'Yes' : 'No'}</td>
                  <td>{item.last_used_at ?? '—'}</td>
                  <td>{item.last_error ?? '—'}</td>
                  <td>
                    <div className="row-actions">
                      <button type="button" className="secondary-button" onClick={() => void handleProbe(item.id)}>Probe</button>
                      <button type="button" className="danger-button" onClick={() => void handleDelete(item.id)}>Delete</button>
                    </div>
                  </td>
                </tr>
              ))
            ) : (
              <tr>
                <td colSpan={7} className="muted">
                  No keys found.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </section>
  );
}
