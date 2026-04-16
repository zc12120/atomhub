import { useEffect, useState } from 'react';
import { api, type AdminModel } from '../api';

const numberFormatter = new Intl.NumberFormat();

interface ModelsPageProps {
  data?: {
    items: AdminModel[];
  };
}

export default function ModelsPage({ data }: ModelsPageProps): JSX.Element {
  const [items, setItems] = useState<AdminModel[]>(data?.items ?? []);
  const [loading, setLoading] = useState(data === undefined);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (data) {
      setItems(data.items);
      setLoading(false);
      return;
    }

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
        const message = loadError instanceof Error ? loadError.message : '加载模型列表失败。';
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
        <h2>模型</h2>
      </header>

      {loading ? <p className="muted">正在加载模型列表…</p> : null}
      {error ? <p className="error-text">{error}</p> : null}

      <div className="table-card">
        <table>
          <thead>
            <tr>
              <th>模型</th>
              <th>提供商</th>
              <th>密钥数</th>
              <th>可用密钥数</th>
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
                  暂无模型数据。
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </section>
  );
}
