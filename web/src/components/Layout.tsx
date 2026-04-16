import { NavLink, Outlet } from 'react-router-dom';

interface LayoutProps {
  username?: string;
  onLogout: () => Promise<void>;
}

const navItems = [
  { to: '/dashboard', label: 'Dashboard' },
  { to: '/keys', label: 'Keys' },
  { to: '/models', label: 'Models' },
  { to: '/health', label: 'Health' }
];

export default function Layout({ username, onLogout }: LayoutProps): JSX.Element {
  const handleLogout = (): void => {
    void onLogout();
  };

  return (
    <div className="shell">
      <header className="topbar">
        <div>
          <h1>AtomHub Admin</h1>
          <p className="topbar-subtitle">Signed in{username ? ` as ${username}` : ''}</p>
        </div>
        <button type="button" className="secondary-button" onClick={handleLogout}>
          Log out
        </button>
      </header>

      <div className="content-grid">
        <aside className="sidebar" aria-label="Admin navigation">
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
