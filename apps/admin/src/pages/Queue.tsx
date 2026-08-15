import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { listCases } from '../api/client';
import type { ModerationCase } from '../types';

export default function Queue() {
  const [cases, setCases] = useState<ModerationCase[]>([]);
  const [statusFilter, setStatusFilter] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [nextCursor, setNextCursor] = useState<string | null>(null);

  const fetchCases = async (cursor?: string | null) => {
    setLoading(true);
    setError('');
    try {
      const page = await listCases(statusFilter || undefined, cursor);
      setCases((prev) => (cursor ? [...prev, ...page.data] : page.data));
      setNextCursor(page.pagination.nextCursor);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load cases');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    setCases([]);
    fetchCases();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [statusFilter]);

  return (
    <div className="queue-container">
      <div className="queue-header">
        <h2>Moderation Queue</h2>
        <select
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value)}
          aria-label="Filter by status"
        >
          <option value="">All statuses</option>
          <option value="open">Open</option>
          <option value="underReview">Under Review</option>
          <option value="resolved">Resolved</option>
        </select>
      </div>

      {error && <div className="error">{error}</div>}

      {loading && cases.length === 0 ? (
        <p>Loading cases...</p>
      ) : (
        <>
          <table className="case-table">
            <thead>
              <tr>
                <th>Target</th>
                <th>Status</th>
                <th>Outcome</th>
                <th>Created</th>
                <th>Reports</th>
              </tr>
            </thead>
            <tbody>
              {cases.map((c) => (
                <tr key={c.id}>
                  <td>
                    <Link to={`/cases/${c.id}`}>
                      {c.targetType} / {c.targetId.slice(0, 8)}
                    </Link>
                  </td>
                  <td>
                    <span className={`badge ${c.status}`}>{c.status}</span>
                  </td>
                  <td>{c.outcome || '-'}</td>
                  <td>{new Date(c.createdAt).toLocaleString()}</td>
                  <td>{c.reportIds.length}</td>
                </tr>
              ))}
              {cases.length === 0 && !loading && (
                <tr>
                  <td colSpan={5}>No cases found.</td>
                </tr>
              )}
            </tbody>
          </table>

          {nextCursor && (
            <button
              className="load-more"
              onClick={() => fetchCases(nextCursor)}
              disabled={loading}
            >
              {loading ? 'Loading...' : 'Load more'}
            </button>
          )}
        </>
      )}
    </div>
  );
}
