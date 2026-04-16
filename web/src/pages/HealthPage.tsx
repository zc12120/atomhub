import { useEffect, useState } from 'react';
import { api, type HealthResponse } from '../api';
import StatCard from '../components/StatCard';

interface HealthPageProps {
  data?: HealthResponse;
}

export default function HealthPage({ data }: HealthPageProps): JSX.Element {
  const [health, setHealth] = useState<HealthResponse | null>(data ?? null);
  const [loading, setLoading] = useState(data === undefined);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (data) {
      setHealth(data);
      setLoading(false);
      return;
    }

    let cancelled = false;

    const load = async (): Promise<void> => {
      setLoading(true);
      setError(null);
      try {
        const response = await api.getHealth();
        if (cancelled) {
          return;
        }
        setHealth(response);
      } catch (loadError) {
        if (cancelled) {
          return;
        }
        const message = loadError instanceof Error ? loadError.message : '加载健康状态失败。';
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
  }, [data]);

  return (
    <section className="page-section">
      <header className="page-header">
        <h2>健康状态</h2>
      </header>

      {loading ? <p className="muted">正在加载健康状态…</p> : null}
      {error ? <p className="error-text">{error}</p> : null}

      {health ? (
        <>
          <div className="stats-grid">
            <StatCard label="健康密钥" value={health.summary.healthy_keys} />
            <StatCard label="异常密钥" value={health.summary.unhealthy_keys} />
            <StatCard label="密钥总数" value={health.summary.total_keys} />
          </div>

          <div className="table-card">
            <table>
              <thead>
                <tr>
                  <th>名称</th>
                  <th>提供商</th>
                  <th>状态</th>
                  <th>最近错误</th>
                </tr>
              </thead>
              <tbody>
                {health.keys.length > 0 ? (
                  health.keys.map((key) => (
                    <tr key={key.id}>
                      <td>{key.label}</td>
                      <td>{key.provider}</td>
                      <td>{key.status}</td>
                      <td>{key.last_error ?? '—'}</td>
                    </tr>
                  ))
                ) : (
                  <tr>
                    <td colSpan={4} className="muted">
                      暂无密钥健康状态记录。
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
