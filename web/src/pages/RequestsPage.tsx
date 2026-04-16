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
        const message = loadError instanceof Error ? loadError.message : 'Failed to load recent requests.';
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
        <h2>Requests</h2>
      </header>

      <div className="card filter-card">
        <label htmlFor="request-model-filter">
          Model Filter
          <select
            id="request-model-filter"
            value={selectedModel}
            onChange={(event) => setSelectedModel(event.target.value)}
          >
            <option value="">All models</option>
            {response?.filters.models.map((model) => (
              <option key={model} value={model}>
                {model}
              </option>
            ))}
          </select>
        </label>
      </div>

      {loading ? <p className="muted">Loading recent requests…</p> : null}
      {error ? <p className="error-text">{error}</p> : null}

      {summary ? (
        <>
          <div className="stats-grid">
            <StatCard label="Requests" value={summary.request_count} />
            <StatCard label="Errors" value={summary.error_count} />
            <StatCard label="Prompt Tokens" value={summary.prompt_tokens} />
            <StatCard label="Completion Tokens" value={summary.completion_tokens} />
            <StatCard label="Total Tokens" value={summary.total_tokens} />
          </div>

          <div className="table-card">
            <table>
              <thead>
                <tr>
                  <th>Model</th>
                  <th>Requests</th>
                  <th>Total Tokens</th>
                  <th>Usage Share</th>
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
                      No requests found for the current filter.
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
                  <th>Time</th>
                  <th>Model</th>
                  <th>Key</th>
                  <th>Provider</th>
                  <th>Status</th>
                  <th>Latency</th>
                  <th>Total Tokens</th>
                  <th>Error</th>
                </tr>
              </thead>
              <tbody>
                {items.length > 0 ? (
                  items.map((item) => (
                    <tr key={item.id}>
                      <td>{item.created_at}</td>
                      <td>{item.model}</td>
                      <td>{item.key_label ?? `#${item.key_id}`}</td>
                      <td>{item.provider ?? '—'}</td>
                      <td>{item.status}</td>
                      <td>{formatNumber(item.latency_ms)} ms</td>
                      <td>{formatNumber(item.total_tokens)}</td>
                      <td>{item.error_message ?? '—'}</td>
                    </tr>
                  ))
                ) : (
                  <tr>
                    <td colSpan={8} className="muted">
                      No requests found for the current filter.
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
