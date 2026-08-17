import { useEffect, useRef, useState, type FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { login, clearSession } from '../api/client';

declare global {
  interface Window {
    turnstile?: {
      render: (
        container: string,
        opts: { sitekey: string; callback: (token: string) => void; 'error-callback': () => void }
      ) => string;
      reset: (widget?: string) => void;
    };
  }
}

const TURNSTILE_SITE_KEY =
  (import.meta as { env?: Record<string, string> }).env?.VITE_TURNSTILE_SITE_KEY ?? '1x00000000000000000000AA';

const WIDGET_ID = 'cf-turnstile';

export default function Login() {
  const navigate = useNavigate();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [token, setToken] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const widgetRef = useRef<HTMLDivElement>(null);
  const widgetId = useRef<string | null>(null);

  useEffect(() => {
    if (!widgetRef.current || !window.turnstile) return;
    widgetId.current = window.turnstile.render(WIDGET_ID, {
      sitekey: TURNSTILE_SITE_KEY,
      callback: (t: string) => setToken(t),
      'error-callback': () => setToken(''),
    });
  }, []);

  useEffect(() => {
    if (window.turnstile) return;
    const script = document.createElement('script');
    script.src = 'https://challenges.cloudflare.com/turnstile/v0/api.js';
    script.async = true;
    document.head.appendChild(script);
  }, []);

  const resetChallenge = () => {
    setToken('');
    if (widgetId.current && window.turnstile) {
      window.turnstile.reset(widgetId.current);
    }
  };

  const handleLogin = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    if (!token) {
      setError('Please complete the human verification.');
      return;
    }
    setLoading(true);
    try {
      const session = await login(username, password, token);
      if (!session.isStaff) {
        clearSession();
        setError('This account does not have staff access.');
        resetChallenge();
        return;
      }
      navigate('/', { replace: true });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Login failed');
      resetChallenge();
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="login-container">
      <div className="login-card">
        <h2>Staff Sign In</h2>
        <p className="hint">Enter your staff username and password.</p>
        {error && <div className="error">{error}</div>}
        <form onSubmit={handleLogin}>
          <label htmlFor="username">Username</label>
          <input
            id="username"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            required
            autoFocus
          />
          <label htmlFor="password">Password</label>
          <input
            id="password"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
          />
          <div ref={widgetRef} id={WIDGET_ID} />
          <button type="submit" disabled={loading}>
            {loading ? 'Signing in...' : 'Sign In'}
          </button>
        </form>
      </div>
    </div>
  );
}