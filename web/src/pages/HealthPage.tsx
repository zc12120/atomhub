import { useEffect, useState } from 'react';
import { api, type HealthResponse } from '../api';
import StatCard from '../components/StatCard';

export default function HealthPage(): JSX.Element {
  const [health, setHealth] = useState<HealthResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
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
        const message = loadError instanceof Error ? loadError.message : 'Failed to load health summary.';
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
  }, []);

  return (
    <section className="page-section">
      <header className="page-header">
        <h2>Health</h2>
      </header>

      {loading ? <p className="muted">Loading health…</p> : null}
      {error ? <p className="error-text">{error}</p> : null}

      {health ? (
        <>
          <div className="stats-grid">
            <StatCard label="Healthy Keys" value={health.summary.healthy_keys} />
            <StatCard label="Unhealthy Keys" value={health.summary.unhealthy_keys} />
            <StatCard label="Total Keys" value={health.summary.total_keys} />
          </div>

          <div className="table-card">
            <table>
              <thead>
                <tr>
                  <th>Label</th>
                  <th>Provider</th>
                  <th>Status</th>
                  <th>Last Error</th>
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
                      No key health records available.
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
