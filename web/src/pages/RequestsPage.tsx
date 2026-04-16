import { useEffect, useMemo, useState } from 'react';
import { api, type RequestsResponse } from '../api';
import StatCard from '../components/StatCard';

interface RequestsPageProps {
  data?: RequestsResponse;
}

interface ModelAggregate {
  model: string;
  requestCount: number;
  totalTokens: number;
}

const numberFormatter = new Intl.NumberFormat();

function formatNumber(value: number): string {
  return numberFormatter.format(value);
}

function formatProvider(provider?: string): string {
  switch (provider) {
    case 'openai':
      return 'OpenAI';
    case 'anthropic':
      return 'Anthropic';
    case 'gemini':
      return 'Gemini';
    case undefined:
    case '':
      return '—';
    default:
      return provider;
  }
}

function formatRequestStatus(status: string): string {
  switch (status) {
    case 'ok':
      return '成功';
    case 'error':
      return '错误';
    default:
      return status;
  }
}

export default function RequestsPage({ data }: RequestsPageProps): JSX.Element {
  const [response, setResponse] = useState<RequestsResponse | null>(data ?? null);
  const [loading, setLoading] = useState(data === undefined);
  const [error, setError] = useState<string | null>(null);
  const [selectedModel, setSelectedModel] = useState(data?.filters.model ?? '');

  useEffect(() => {
    if (data) {
      setResponse(data);
      setLoading(false);
      return;
    }

    let cancelled = false;

    const load = async (): Promise<void> => {
      setLoading(true);
      setError(null);
      try {
        const next = await api.getRequests(selectedModel || undefined);
        if (cancelled) {
          return;
        }
        setResponse(next);
      } catch (loadError) {
        if (cancelled) {
          return;
        }
        const message = loadError instanceof Error ? loadError.message : '加载请求记录失败。';
        setError(message);
      } finally {
        if (cancelled === false) {
          setLoading(false);
        }
      }
    };

    void load();

    return () => {
      cancelled = true;
    };
  }, [data, selectedModel]);

  const items = useMemo(() => {
    if (!response) {
      return [];
    }
    if (!selectedModel) {
      return response.items;
    }
    return response.items.filter((item) => item.model === selectedModel);
  }, [response, selectedModel]);

  const summary = useMemo(() => {
    if (!response) {
      return null;
    }
    if (!selectedModel) {
      return response.summary;
    }
    return items.reduce(
      (acc, item) => {
        acc.request_count += 1;
        acc.prompt_tokens += item.prompt_tokens;
        acc.completion_tokens += item.completion_tokens;
        acc.total_tokens += item.total_tokens;
        if (item.status !== 'ok') {
          acc.error_count += 1;
        }
        return acc;
      },
      { request_count: 0, error_count: 0, prompt_tokens: 0, completion_tokens: 0, total_tokens: 0 }
    );
  }, [response, selectedModel, items]);

  const modelAggregates = useMemo<ModelAggregate[]>(() => {
    const bucket = new Map<string, ModelAggregate>();
    for (const item of items) {
      const current = bucket.get(item.model) ?? { model: item.model, requestCount: 0, totalTokens: 0 };
      current.requestCount += 1;
      current.totalTokens += item.total_tokens;
      bucket.set(item.model, current);
    }
    return Array.from(bucket.values()).sort((left, right) => right.totalTokens - left.totalTokens || left.model.localeCompare(right.model));
  }, [items]);

  const maxTokens = modelAggregates.reduce((max, item) => Math.max(max, item.totalTokens), 0);

  return (
    <section className="page-section">
      <header className="page-header">
        <h2>请求记录</h2>
      </header>

      <div className="card filter-card">
        <label htmlFor="request-model-filter">
          模型筛选
          <select
            id="request-model-filter"
            value={selectedModel}
            onChange={(event) => setSelectedModel(event.target.value)}
          >
            <option value="">全部模型</option>
            {response?.filters.models.map((model) => (
              <option key={model} value={model}>
                {model}
              </option>
            ))}
          </select>
        </label>
      </div>

      {loading ? <p className="muted">正在加载最近请求…</p> : null}
      {error ? <p className="error-text">{error}</p> : null}

      {summary ? (
        <>
          <div className="stats-grid">
            <StatCard label="请求数" value={summary.request_count} />
            <StatCard label="错误数" value={summary.error_count} />
            <StatCard label="Prompt token" value={summary.prompt_tokens} />
            <StatCard label="Completion token" value={summary.completion_tokens} />
            <StatCard label="Total token" value={summary.total_tokens} />
          </div>

          <div className="table-card">
            <table>
              <thead>
                <tr>
                  <th>模型</th>
                  <th>请求数</th>
                  <th>Total token</th>
                  <th>占比</th>
                </tr>
              </thead>
              <tbody>
                {modelAggregates.length > 0 ? (
                  modelAggregates.map((item) => {
                    const ratio = maxTokens === 0 ? 0 : (item.totalTokens / maxTokens) * 100;
                    return (
                      <tr key={item.model}>
                        <td>{item.model}</td>
                        <td>{formatNumber(item.requestCount)}</td>
                        <td>{formatNumber(item.totalTokens)}</td>
                        <td>
                          <div className="usage-bar-shell">
                            <div className="usage-bar-fill" style={{ width: `${ratio}%` }} />
                          </div>
                        </td>
                      </tr>
                    );
                  })
                ) : (
                  <tr>
                    <td colSpan={4} className="muted">
                      没有符合当前筛选条件的请求记录。
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>

          <div className="table-card">
            <table>
              <thead>
                <tr>
                  <th>时间</th>
                  <th>模型</th>
                  <th>密钥</th>
                  <th>提供商</th>
                  <th>状态</th>
                  <th>延迟</th>
                  <th>Total token</th>
                  <th>错误</th>
                </tr>
              </thead>
              <tbody>
                {items.length > 0 ? (
                  items.map((item) => (
                    <tr key={item.id}>
                      <td>{item.created_at}</td>
                      <td>{item.model}</td>
                      <td>{item.key_label ?? `#${item.key_id}`}</td>
                      <td>{formatProvider(item.provider)}</td>
                      <td>{formatRequestStatus(item.status)}</td>
                      <td>{formatNumber(item.latency_ms)} ms</td>
                      <td>{formatNumber(item.total_tokens)}</td>
                      <td>{item.error_message ?? '—'}</td>
                    </tr>
                  ))
                ) : (
                  <tr>
                    <td colSpan={8} className="muted">
                      没有符合当前筛选条件的请求记录。
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </>
      ) : null}
    </section>
  );
}
