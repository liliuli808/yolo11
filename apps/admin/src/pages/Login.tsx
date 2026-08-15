import { useState, type FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { sendEmailCode, createEmailSession, clearSession } from '../api/client';

export default function Login() {
  const navigate = useNavigate();
  const [email, setEmail] = useState('');
  const [code, setCode] = useState('');
  const [codeSent, setCodeSent] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const handleRequestCode = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      await sendEmailCode(email);
      setCodeSent(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to send code');
    } finally {
      setLoading(false);
    }
  };

  const handleLogin = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      const session = await createEmailSession(email, code);
      if (!session.isStaff) {
        clearSession();
        setError('This account does not have staff access.');
        return;
      }
      navigate('/', { replace: true });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Login failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="login-container">
      <div className="login-card">
        <h2>Staff Sign In</h2>
        <p className="hint">
          Enter a staff email address. A one-time code will be sent to your inbox.
        </p>
        {error && <div className="error">{error}</div>}
        {!codeSent ? (
          <form onSubmit={handleRequestCode}>
            <label htmlFor="email">Email</label>
            <input
              id="email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
              autoFocus
            />
            <button type="submit" disabled={loading}>
              {loading ? 'Sending...' : 'Send Code'}
            </button>
          </form>
        ) : (
          <form onSubmit={handleLogin}>
            <label htmlFor="code">Verification Code</label>
            <input
              id="code"
              type="text"
              inputMode="numeric"
              maxLength={6}
              value={code}
              onChange={(e) => setCode(e.target.value)}
              required
              autoFocus
            />
            <button type="submit" disabled={loading}>
              {loading ? 'Signing in...' : 'Sign In'}
            </button>
            <button
              type="button"
              className="secondary"
              onClick={() => setCodeSent(false)}
              disabled={loading}
            >
              Use a different email
            </button>
          </form>
        )}
      </div>
    </div>
  );
}
