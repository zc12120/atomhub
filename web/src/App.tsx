import { useEffect, useState } from 'react';
import { Navigate, Route, Routes } from 'react-router-dom';
import Layout from './components/Layout';
import { fetchSession, isAuthenticated, logoutAndClearSession, type SessionState } from './auth';
import DashboardPage from './pages/DashboardPage';
import HealthPage from './pages/HealthPage';
import KeysPage from './pages/KeysPage';
import LoginPage from './pages/LoginPage';
import ModelsPage from './pages/ModelsPage';
import RequestsPage from './pages/RequestsPage';

export default function App(): JSX.Element {
  const [session, setSession] = useState<SessionState | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;

    const load = async (): Promise<void> => {
      setLoading(true);
      const nextSession = await fetchSession();
      if (cancelled) {
        return;
      }
      setSession(nextSession);
      setLoading(false);
    };

    void load();

    return () => {
      cancelled = true;
    };
  }, []);

  const handleLogout = async (): Promise<void> => {
    const guest = await logoutAndClearSession();
    setSession(guest);
  };

  if (loading) {
    return (
      <div className="app-loading">
        <p className="muted">Checking admin session…</p>
      </div>
    );
  }

  const hasSession = isAuthenticated(session);

  return (
    <Routes>
      <Route path="/login" element={<LoginPage session={session} onLogin={setSession} />} />
      <Route
        path="/"
        element={
          hasSession ? (
            <Layout username={session?.username} onLogout={handleLogout} />
          ) : (
            <Navigate to="/login" replace />
          )
        }
      >
        <Route index element={<Navigate to="/dashboard" replace />} />
        <Route path="dashboard" element={<DashboardPage />} />
        <Route path="keys" element={<KeysPage />} />
        <Route path="models" element={<ModelsPage />} />
        <Route path="requests" element={<RequestsPage />} />
        <Route path="health" element={<HealthPage />} />
      </Route>
      <Route path="*" element={<Navigate to={hasSession ? '/dashboard' : '/login'} replace />} />
    </Routes>
  );
}
