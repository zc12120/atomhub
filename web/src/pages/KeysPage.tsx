import { Fragment, FormEvent, useEffect, useState } from 'react';
import { api, type AdminKey, type CreateKeyPayload, type UpdateKeyPayload } from '../api';

const emptyForm: CreateKeyPayload = {
  name: '',
  provider: 'openai',
  base_url: '',
  api_key: '',
  enabled: true
};

function formatProvider(provider: string): string {
  switch (provider) {
    case 'openai':
      return 'OpenAI';
    case 'anthropic':
      return 'Anthropic';
    case 'gemini':
      return 'Gemini';
    default:
      return provider;
  }
}

function formatKeyStatus(status: string): string {
  switch (status) {
    case 'healthy':
      return '正常';
    case 'degraded':
      return '异常';
    case 'cooling_down':
      return '冷却中';
    case 'disabled':
      return '已停用';
    default:
      return status;
  }
}

export default function KeysPage(): JSX.Element {
  const [items, setItems] = useState<AdminKey[]>([]);
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [form, setForm] = useState<CreateKeyPayload>(emptyForm);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [editForm, setEditForm] = useState<UpdateKeyPayload>({ name: '', provider: 'openai', base_url: '', api_key: '' });

  const loadKeys = async (): Promise<void> => {
    setLoading(true);
    setError(null);
    try {
      const response = await api.getKeys();
      setItems(response.items);
    } catch (loadError) {
      const message = loadError instanceof Error ? loadError.message : '加载密钥列表失败。';
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
      const message = submitError instanceof Error ? submitError.message : '创建密钥失败。';
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
      const message = probeError instanceof Error ? probeError.message : '探测密钥失败。';
      setError(message);
    }
  };

  const handleDelete = async (id: number): Promise<void> => {
    setError(null);
    try {
      await api.deleteKey(id);
      await loadKeys();
    } catch (deleteError) {
      const message = deleteError instanceof Error ? deleteError.message : '删除密钥失败。';
      setError(message);
    }
  };

  const handleToggleEnabled = async (item: AdminKey): Promise<void> => {
    setError(null);
    try {
      await api.updateKey(item.id, { enabled: !item.enabled });
      if (editingId === item.id) {
        setEditingId(null);
      }
      await loadKeys();
    } catch (updateError) {
      const message = updateError instanceof Error ? updateError.message : '更新密钥失败。';
      setError(message);
    }
  };

  const handleStartEdit = (item: AdminKey): void => {
    setEditingId(item.id);
    setEditForm({
      name: item.label,
      provider: item.provider,
      base_url: item.base_url ?? '',
      api_key: ''
    });
  };

  const handleSaveEdit = async (id: number): Promise<void> => {
    setError(null);
    try {
      const payload: UpdateKeyPayload = {
        name: editForm.name?.trim(),
        provider: editForm.provider,
        base_url: editForm.base_url?.trim() ?? ''
      };
      if (editForm.api_key?.trim()) {
        payload.api_key = editForm.api_key.trim();
      }
      await api.updateKey(id, payload);
      setEditingId(null);
      setEditForm({ name: '', provider: 'openai', base_url: '', api_key: '' });
      await loadKeys();
    } catch (updateError) {
      const message = updateError instanceof Error ? updateError.message : '更新密钥失败。';
      setError(message);
    }
  };

  return (
    <section className="page-section">
      <header className="page-header">
        <h2>密钥</h2>
      </header>

      <form className="card form-grid" onSubmit={handleSubmit}>
        <h3>新增上游密钥</h3>
        <label>
          名称
          <input value={form.name} onChange={(event) => setForm((current) => ({ ...current, name: event.target.value }))} required />
        </label>
        <label>
          提供商
          <select value={form.provider} onChange={(event) => setForm((current) => ({ ...current, provider: event.target.value }))}>
            <option value="openai">OpenAI</option>
            <option value="anthropic">Anthropic</option>
            <option value="gemini">Gemini</option>
          </select>
        </label>
        <label>
          Base URL
          <input value={form.base_url} onChange={(event) => setForm((current) => ({ ...current, base_url: event.target.value }))} placeholder="可选" />
        </label>
        <label className="full-width">
          API Key
          <input value={form.api_key} onChange={(event) => setForm((current) => ({ ...current, api_key: event.target.value }))} required />
        </label>
        <button type="submit" disabled={submitting}>{submitting ? '保存中…' : '保存密钥'}</button>
      </form>

      {loading ? <p className="muted">正在加载密钥列表…</p> : null}
      {error ? <p className="error-text">{error}</p> : null}

      <div className="table-card">
        <table>
          <thead>
            <tr>
              <th>名称</th>
              <th>提供商</th>
              <th>状态</th>
              <th>启用</th>
              <th>最近使用</th>
              <th>最近错误</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            {items.length > 0 ? (
              items.map((item) => (
                <Fragment key={item.id}>
                  <tr>
                    <td>{item.label}</td>
                    <td>{formatProvider(item.provider)}</td>
                    <td>{formatKeyStatus(item.status)}</td>
                    <td>{item.enabled ? '是' : '否'}</td>
                    <td>{item.last_used_at ?? '—'}</td>
                    <td>{item.last_error ?? '—'}</td>
                    <td>
                      <div className="row-actions">
                        <button type="button" className="secondary-button" onClick={() => handleStartEdit(item)}>编辑</button>
                        <button type="button" className="secondary-button" onClick={() => void handleToggleEnabled(item)}>
                          {item.enabled ? '停用' : '启用'}
                        </button>
                        <button type="button" className="secondary-button" onClick={() => void handleProbe(item.id)}>探测</button>
                        <button type="button" className="danger-button" onClick={() => void handleDelete(item.id)}>删除</button>
                      </div>
                    </td>
                  </tr>
                  {editingId === item.id ? (
                    <tr>
                      <td colSpan={7}>
                        <div className="inline-editor">
                          <label>
                            编辑名称
                            <input
                              value={editForm.name ?? ''}
                              onChange={(event) => setEditForm((current) => ({ ...current, name: event.target.value }))}
                            />
                          </label>
                          <label>
                            编辑提供商
                            <select
                              value={editForm.provider ?? 'openai'}
                              onChange={(event) => setEditForm((current) => ({ ...current, provider: event.target.value }))}
                            >
                              <option value="openai">OpenAI</option>
                              <option value="anthropic">Anthropic</option>
                              <option value="gemini">Gemini</option>
                            </select>
                          </label>
                          <label>
                            编辑 Base URL
                            <input
                              value={editForm.base_url ?? ''}
                              onChange={(event) => setEditForm((current) => ({ ...current, base_url: event.target.value }))}
                            />
                          </label>
                          <label>
                            新的 API Key
                            <input
                              value={editForm.api_key ?? ''}
                              onChange={(event) => setEditForm((current) => ({ ...current, api_key: event.target.value }))}
                            />
                          </label>
                          <div className="row-actions">
                            <button type="button" onClick={() => void handleSaveEdit(item.id)}>保存修改</button>
                            <button
                              type="button"
                              className="secondary-button"
                              onClick={() => {
                                setEditingId(null);
                                setEditForm({ name: '', provider: 'openai', base_url: '', api_key: '' });
                              }}
                            >
                              取消
                            </button>
                          </div>
                        </div>
                      </td>
                    </tr>
                  ) : null}
                </Fragment>
              ))
            ) : (
              <tr>
                <td colSpan={7} className="muted">
                  暂无密钥。
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </section>
  );
}
