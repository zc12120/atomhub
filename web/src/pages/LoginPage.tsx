import { FormEvent, useState } from 'react';
import { Navigate } from 'react-router-dom';
import { isAuthenticated, loginAndFetchSession, type SessionState } from '../auth';

interface LoginPageProps {
  session: SessionState | null;
  onLogin: (session: SessionState) => void;
}

export default function LoginPage({ session, onLogin }: LoginPageProps): JSX.Element {
  const [username, setUsername] = useState('admin');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  if (isAuthenticated(session)) {
    return <Navigate to="/dashboard" replace />;
  }

  const handleSubmit = async (event: FormEvent<HTMLFormElement>): Promise<void> => {
    event.preventDefault();
    setSubmitting(true);
    setError(null);

    try {
      const nextSession = await loginAndFetchSession(username, password);
      if (nextSession.authenticated === false) {
        setError('Login succeeded but no active session was found.');
        return;
      }
      onLogin(nextSession);
    } catch (submitError) {
      const message = submitError instanceof Error ? submitError.message : 'Login failed.';
      setError(message);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="login-shell">
      <form className="card" onSubmit={handleSubmit}>
        <h1>Admin Login</h1>
        <p className="muted">Sign in to view dashboard metrics and key health.</p>

        <label htmlFor="username">Username</label>
        <input
          id="username"
          name="username"
          autoComplete="username"
          value={username}
          onChange={(event) => setUsername(event.target.value)}
          required
        />

        <label htmlFor="password">Password</label>
        <input
          id="password"
          type="password"
          name="password"
          autoComplete="current-password"
          value={password}
          onChange={(event) => setPassword(event.target.value)}
          required
        />

        {error ? <p className="error-text">{error}</p> : null}

        <button type="submit" disabled={submitting}>
          {submitting ? 'Signing in…' : 'Sign in'}
        </button>
      </form>
    </div>
  );
}
