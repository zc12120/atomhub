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
        setError('登录成功，但未检测到有效会话。');
        return;
      }
      onLogin(nextSession);
    } catch (submitError) {
      const fallbackMessage = '登录失败。';
      if (submitError instanceof Error) {
        const normalized = submitError.message.trim().toLowerCase();
        if (
          normalized === 'invalid credentials' ||
          normalized === 'unauthorized' ||
          normalized.startsWith('request failed: 401')
        ) {
          setError('用户名或密码错误。');
        } else if (submitError.message.trim()) {
          setError(submitError.message);
        } else {
          setError(fallbackMessage);
        }
      } else {
        setError(fallbackMessage);
      }
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="login-shell">
      <form className="card" onSubmit={handleSubmit}>
        <h1>管理员登录</h1>
        <p className="muted">登录后可查看仪表盘、密钥状态与请求记录。</p>

        <label htmlFor="username">用户名</label>
        <input
          id="username"
          name="username"
          autoComplete="username"
          value={username}
          onChange={(event) => setUsername(event.target.value)}
          required
        />

        <label htmlFor="password">密码</label>
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
          {submitting ? '登录中…' : '登录'}
        </button>
      </form>
    </div>
  );
}
