import { NavLink, Outlet } from 'react-router-dom';

interface LayoutProps {
  username?: string;
  onLogout: () => Promise<void>;
}

const navItems = [
  { to: '/dashboard', label: '仪表盘' },
  { to: '/keys', label: '密钥' },
  { to: '/downstream-keys', label: '下游密钥' },
  { to: '/models', label: '模型' },
  { to: '/requests', label: '请求记录' },
  { to: '/health', label: '健康状态' }
];

export default function Layout({ username, onLogout }: LayoutProps): JSX.Element {
  const handleLogout = (): void => {
    void onLogout();
  };

  return (
    <div className="shell">
      <header className="topbar">
        <div>
          <h1>AtomHub 管理后台</h1>
          <p className="topbar-subtitle">已登录{username ? `：${username}` : ''}</p>
        </div>
        <button type="button" className="secondary-button" onClick={handleLogout}>
          退出登录
        </button>
      </header>

      <div className="content-grid">
        <aside className="sidebar" aria-label="管理后台导航">
          <nav>
            <ul>
              {navItems.map((item) => (
                <li key={item.to}>
                  <NavLink
                    to={item.to}
                    className={({ isActive }) => (isActive ? 'nav-link active' : 'nav-link')}
                  >
                    {item.label}
                  </NavLink>
                </li>
              ))}
            </ul>
          </nav>
        </aside>

        <main className="main-panel">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
