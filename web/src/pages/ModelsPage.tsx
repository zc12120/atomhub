import { useEffect, useState } from 'react';
import { api, type AdminModel } from '../api';

const numberFormatter = new Intl.NumberFormat();

export default function ModelsPage(): JSX.Element {
  const [items, setItems] = useState<AdminModel[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    const load = async (): Promise<void> => {
      setLoading(true);
      setError(null);
      try {
        const response = await api.getModels();
        if (cancelled) {
          return;
        }
        setItems(response.items);
      } catch (loadError) {
        if (cancelled) {
          return;
        }
        const message = loadError instanceof Error ? loadError.message : 'Failed to load models.';
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
        <h2>Models</h2>
      </header>

      {loading ? <p className="muted">Loading models…</p> : null}
      {error ? <p className="error-text">{error}</p> : null}

      <div className="table-card">
        <table>
          <thead>
            <tr>
              <th>Model</th>
              <th>Provider</th>
              <th>Keys</th>
              <th>Healthy Keys</th>
            </tr>
          </thead>
          <tbody>
            {items.length > 0 ? (
              items.map((item) => (
                <tr key={item.provider + ':' + item.model}>
                  <td>{item.model}</td>
                  <td>{item.provider}</td>
                  <td>{numberFormatter.format(item.key_count)}</td>
                  <td>{numberFormatter.format(item.healthy_keys)}</td>
                </tr>
              ))
            ) : (
              <tr>
                <td colSpan={4} className="muted">
                  No models found.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </section>
  );
}
