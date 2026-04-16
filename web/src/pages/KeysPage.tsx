import { useEffect, useState } from 'react';
import { api, type AdminKey } from '../api';

export default function KeysPage(): JSX.Element {
  const [items, setItems] = useState<AdminKey[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    const load = async (): Promise<void> => {
      setLoading(true);
      setError(null);
      try {
        const response = await api.getKeys();
        if (cancelled) {
          return;
        }
        setItems(response.items);
      } catch (loadError) {
        if (cancelled) {
          return;
        }
        const message = loadError instanceof Error ? loadError.message : 'Failed to load keys.';
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
        <h2>Keys</h2>
      </header>

      {loading ? <p className="muted">Loading keys…</p> : null}
      {error ? <p className="error-text">{error}</p> : null}

      <div className="table-card">
        <table>
          <thead>
            <tr>
              <th>Label</th>
              <th>Provider</th>
              <th>Status</th>
              <th>Last Used</th>
              <th>Last Error</th>
            </tr>
          </thead>
          <tbody>
            {items.length > 0 ? (
              items.map((item) => (
                <tr key={item.id}>
                  <td>{item.label}</td>
                  <td>{item.provider}</td>
                  <td>{item.status}</td>
                  <td>{item.last_used_at ?? '—'}</td>
                  <td>{item.last_error ?? '—'}</td>
                </tr>
              ))
            ) : (
              <tr>
                <td colSpan={5} className="muted">
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
