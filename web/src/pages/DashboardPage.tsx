import { useEffect, useState } from 'react';
import { api, type DashboardResponse } from '../api';
import StatCard from '../components/StatCard';

interface DashboardPageProps {
  data?: DashboardResponse;
}

const numberFormatter = new Intl.NumberFormat();

function renderValue(value: number): string {
  return numberFormatter.format(value);
}

export default function DashboardPage({ data }: DashboardPageProps): JSX.Element {
  const [dashboard, setDashboard] = useState<DashboardResponse | null>(data ?? null);
  const [loading, setLoading] = useState(data === undefined);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (data) {
      setDashboard(data);
      setLoading(false);
      return;
    }

    let cancelled = false;

    const loadDashboard = async (): Promise<void> => {
      setLoading(true);
      setError(null);
      try {
        const response = await api.getDashboard();
        if (cancelled) {
          return;
        }
        setDashboard(response);
      } catch (loadError) {
        if (cancelled) {
          return;
        }
        const message = loadError instanceof Error ? loadError.message : '加载仪表盘失败。';
        setError(message);
      } finally {
        if (cancelled === false) {
          setLoading(false);
        }
      }
    };

    void loadDashboard();

    return () => {
      cancelled = true;
    };
  }, [data]);

  return (
    <section className="page-section">
      <header className="page-header">
        <h2>仪表盘</h2>
      </header>

      {loading ? <p className="muted">正在加载用量汇总…</p> : null}
      {error ? <p className="error-text">{error}</p> : null}

      {dashboard ? (
        <>
          <div className="stats-grid">
            <StatCard label="Prompt token" value={dashboard.summary.prompt_tokens} />
            <StatCard label="Completion token" value={dashboard.summary.completion_tokens} />
            <StatCard label="Total token" value={dashboard.summary.total_tokens} />
          </div>

          <div className="table-card">
            <table>
              <thead>
                <tr>
                  <th>模型</th>
                  <th>Prompt token</th>
                  <th>Completion token</th>
                  <th>Total token</th>
                  <th>请求数</th>
                </tr>
              </thead>
              <tbody>
                {dashboard.items.map((item) => (
                  <tr key={item.model}>
                    <td>{item.model}</td>
                    <td>{renderValue(item.prompt_tokens)}</td>
                    <td>{renderValue(item.completion_tokens)}</td>
                    <td>{renderValue(item.total_tokens)}</td>
                    <td>{renderValue(item.request_count)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      ) : null}
    </section>
  );
}
