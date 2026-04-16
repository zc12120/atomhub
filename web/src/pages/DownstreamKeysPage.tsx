import { FormEvent, useEffect, useState } from 'react';
import { api, type CreateDownstreamKeyPayload, type DownstreamKey } from '../api';

const emptyForm: CreateDownstreamKeyPayload = {
  name: '',
  enabled: true
};

const numberFormatter = new Intl.NumberFormat();

function formatDateTime(value?: string | null): string {
  if (!value) {
    return '—';
  }
  return value;
}

export default function DownstreamKeysPage(): JSX.Element {
  const [items, setItems] = useState<DownstreamKey[]>([]);
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [createdToken, setCreatedToken] = useState<string | null>(null);
  const [revealedTokens, setRevealedTokens] = useState<Record<number, string>>({});
  const [copiedKeyID, setCopiedKeyID] = useState<number | null>(null);
  const [form, setForm] = useState<CreateDownstreamKeyPayload>(emptyForm);

  const loadKeys = async (): Promise<void> => {
    setLoading(true);
    setError(null);
    try {
      const response = await api.getDownstreamKeys();
      setItems(response.items);
    } catch (loadError) {
      const message = loadError instanceof Error ? loadError.message : '加载下游密钥失败。';
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
    setCreatedToken(null);
    try {
      const response = await api.createDownstreamKey(form);
      setCreatedToken(response.token);
      setCopiedKeyID(null);
      setForm(emptyForm);
      await loadKeys();
    } catch (submitError) {
      const message = submitError instanceof Error ? submitError.message : '创建下游密钥失败。';
      setError(message);
    } finally {
      setSubmitting(false);
    }
  };

  const handleToggleEnabled = async (item: DownstreamKey): Promise<void> => {
    setError(null);
    try {
      await api.updateDownstreamKey(item.id, { enabled: !item.enabled });
      await loadKeys();
    } catch (updateError) {
      const message = updateError instanceof Error ? updateError.message : '更新下游密钥失败。';
      setError(message);
    }
  };

  const handleDelete = async (id: number): Promise<void> => {
    setError(null);
    try {
      await api.deleteDownstreamKey(id);
      await loadKeys();
    } catch (deleteError) {
      const message = deleteError instanceof Error ? deleteError.message : '删除下游密钥失败。';
      setError(message);
    }
  };

  const handleReveal = async (id: number): Promise<string | null> => {
    setError(null);
    try {
      const response = await api.revealDownstreamKey(id);
      setRevealedTokens((current) => ({ ...current, [id]: response.token }));
      setCopiedKeyID(null);
      return response.token;
    } catch (revealError) {
      const message = revealError instanceof Error ? revealError.message : '查看下游密钥失败。';
      setError(message);
      return null;
    }
  };

  const handleCopy = async (item: DownstreamKey): Promise<void> => {
    setError(null);
    const token = revealedTokens[item.id] ?? (await handleReveal(item.id));
    if (!token) {
      return;
    }
    try {
      await navigator.clipboard.writeText(token);
      setCopiedKeyID(item.id);
    } catch (copyError) {
      const message = copyError instanceof Error ? copyError.message : '复制下游密钥失败。';
      setError(message);
    }
  };

  const handleRegenerate = async (id: number): Promise<void> => {
    setError(null);
    try {
      const response = await api.regenerateDownstreamKey(id);
      setRevealedTokens((current) => ({ ...current, [id]: response.token }));
      setCreatedToken(response.token);
      setCopiedKeyID(null);
      await loadKeys();
    } catch (regenerateError) {
      const message = regenerateError instanceof Error ? regenerateError.message : '重新生成下游密钥失败。';
      setError(message);
    }
  };

  return (
    <section className="page-section">
      <header className="page-header">
        <h2>下游密钥</h2>
      </header>

      <form className="card form-grid" onSubmit={handleSubmit}>
        <h3>新增下游密钥</h3>
        <label>
          名称
          <input
            value={form.name}
            onChange={(event) => setForm((current) => ({ ...current, name: event.target.value }))}
            required
          />
        </label>
        <label>
          创建后立即启用
          <select
            value={form.enabled ? 'true' : 'false'}
            onChange={(event) => setForm((current) => ({ ...current, enabled: event.target.value === 'true' }))}
          >
            <option value="true">是</option>
            <option value="false">否</option>
          </select>
        </label>
        <button type="submit" disabled={submitting}>
          {submitting ? '创建中…' : '创建下游密钥'}
        </button>
      </form>

      {createdToken ? (
        <div className="card token-card">
          <h3>已生成新的下游 token</h3>
          <p className="muted">请立即复制并妥善保存此 token。以后也可以在列表里使用“查看密钥”或“复制密钥”。</p>
          <input value={createdToken} readOnly aria-label="新生成 token" />
        </div>
      ) : null}

      {loading ? <p className="muted">正在加载下游密钥…</p> : null}
      {error ? <p className="error-text">{error}</p> : null}

      <div className="table-card">
        <table>
          <thead>
            <tr>
              <th>名称</th>
              <th>密钥</th>
              <th>状态</th>
              <th>最近使用</th>
              <th>请求数</th>
              <th>Total token</th>
              <th>创建时间</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            {items.length > 0 ? (
              items.map((item) => (
                <tr key={item.id}>
                  <td>{item.name}</td>
                  <td>
                    <div className="token-display-stack">
                      <span>{item.masked_token}</span>
                      {revealedTokens[item.id] ? (
                        <input value={revealedTokens[item.id]} readOnly aria-label={`密钥-${item.id}`} />
                      ) : null}
                      {!item.can_reveal ? <span className="muted">旧版密钥需重新生成后才能查看</span> : null}
                    </div>
                  </td>
                  <td>{item.enabled ? '启用中' : '已停用'}</td>
                  <td>{formatDateTime(item.last_used_at)}</td>
                  <td>{numberFormatter.format(item.request_count)}</td>
                  <td>{numberFormatter.format(item.total_tokens)}</td>
                  <td>{formatDateTime(item.created_at)}</td>
                  <td>
                    <div className="row-actions">
                      <button
                        type="button"
                        className="secondary-button"
                        onClick={() => void handleToggleEnabled(item)}
                      >
                        {item.enabled ? '停用' : '启用'}
                      </button>
                      <button
                        type="button"
                        className="secondary-button"
                        onClick={() => void handleReveal(item.id)}
                        disabled={!item.can_reveal}
                      >
                        查看密钥
                      </button>
                      <button
                        type="button"
                        className="secondary-button"
                        onClick={() => void handleCopy(item)}
                        disabled={!item.can_reveal && !revealedTokens[item.id]}
                      >
                        {copiedKeyID === item.id ? '已复制' : '复制密钥'}
                      </button>
                      <button
                        type="button"
                        className="secondary-button"
                        onClick={() => void handleRegenerate(item.id)}
                      >
                        重新生成
                      </button>
                      <button
                        type="button"
                        className="danger-button"
                        onClick={() => void handleDelete(item.id)}
                      >
                        删除
                      </button>
                    </div>
                  </td>
                </tr>
              ))
            ) : (
              <tr>
                <td colSpan={8} className="muted">
                  还没有创建任何下游密钥。
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </section>
  );
}
