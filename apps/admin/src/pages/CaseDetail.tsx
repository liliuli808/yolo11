import { useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import { getCase, updateCase, listCaseActions } from '../api/client';
import type { ModerationCase, ModerationAction } from '../types';

const outcomes = [
  { value: 'noAction', label: 'No Action' },
  { value: 'warn', label: 'Warn' },
  { value: 'hide', label: 'Hide Content' },
  { value: 'remove', label: 'Remove Content' },
  { value: 'restore', label: 'Restore Content' },
  { value: 'restrictPersona', label: 'Restrict Persona' },
  { value: 'suspendAccount', label: 'Suspend Account' },
  { value: 'banAccount', label: 'Ban Account' },
];

export default function CaseDetail() {
  const { caseId } = useParams<{ caseId: string }>();
  const [c, setCase] = useState<ModerationCase | null>(null);
  const [actions, setActions] = useState<ModerationAction[]>([]);
  const [selectedOutcome, setSelectedOutcome] = useState('');
  const [notes, setNotes] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const fetchData = async () => {
    if (!caseId) return;
    setLoading(true);
    setError('');
    try {
      const [caseData, actionData] = await Promise.all([
        getCase(caseId),
        listCaseActions(caseId),
      ]);
      setCase(caseData);
      setActions(actionData);
      setSelectedOutcome(caseData.outcome || '');
      setNotes(caseData.notes || '');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load case');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [caseId]);

  const handleResolve = async () => {
    if (!caseId || !selectedOutcome) return;
    setError('');
    try {
      const updated = await updateCase(caseId, 'resolved', selectedOutcome, notes);
      setCase(updated);
      setActions((prev) => [
        {
          id: `action-${Date.now()}`,
          moderatorUserId: null,
          actionType: selectedOutcome,
          targetType: updated.targetType,
          targetId: updated.targetId,
          note: notes || null,
          createdAt: new Date().toISOString(),
        },
        ...prev,
      ]);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to resolve case');
    }
  };

  const handleMarkUnderReview = async () => {
    if (!caseId) return;
    setError('');
    try {
      const updated = await updateCase(caseId, 'underReview', undefined, notes);
      setCase(updated);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update case');
    }
  };

  if (loading) return <p>Loading case...</p>;
  if (error && !c) return <div className="error">{error}</div>;
  if (!c) return <p>Case not found.</p>;

  return (
    <div className="case-detail">
      <Link to="/" className="back-link">← Back to queue</Link>
      <h2>Case {c.id.slice(0, 8)}</h2>

      {error && <div className="error">{error}</div>}

      <section className="case-meta">
        <div>
          <strong>Target Type:</strong> {c.targetType}
        </div>
        <div>
          <strong>Target ID:</strong> {c.targetId}
        </div>
        <div>
          <strong>Status:</strong>{' '}
          <span className={`badge ${c.status}`}>{c.status}</span>
        </div>
        {c.outcome && (
          <div>
            <strong>Outcome:</strong> {c.outcome}
          </div>
        )}
        <div>
          <strong>Created:</strong> {new Date(c.createdAt).toLocaleString()}
        </div>
        <div>
          <strong>Reports:</strong> {c.reportIds.length}
        </div>
      </section>

      {c.status !== 'resolved' && (
        <section className="case-actions">
          <h3>Take Action</h3>
          <label htmlFor="outcome">Outcome</label>
          <select
            id="outcome"
            value={selectedOutcome}
            onChange={(e) => setSelectedOutcome(e.target.value)}
          >
            <option value="">Select outcome...</option>
            {outcomes.map((o) => (
              <option key={o.value} value={o.value}>
                {o.label}
              </option>
            ))}
          </select>

          <label htmlFor="notes">Notes</label>
          <textarea
            id="notes"
            rows={4}
            value={notes}
            onChange={(e) => setNotes(e.target.value)}
            placeholder="Optional moderator notes"
          />

          <div className="button-row">
            <button onClick={handleResolve} disabled={!selectedOutcome}>
              Resolve Case
            </button>
            <button className="secondary" onClick={handleMarkUnderReview}>
              Mark Under Review
            </button>
          </div>
        </section>
      )}

      <section className="audit-history">
        <h3>Audit History</h3>
        {actions.length === 0 ? (
          <p>No actions recorded yet.</p>
        ) : (
          <ul>
            {actions.map((a) => (
              <li key={a.id} className="audit-entry">
                <div className="audit-header">
                  <span className="audit-action">{a.actionType}</span>
                  <span className="audit-time">
                    {new Date(a.createdAt).toLocaleString()}
                  </span>
                </div>
                {a.note && <p>{a.note}</p>}
                {a.targetType && a.targetId && (
                  <div className="audit-target">
                    {a.targetType} / {a.targetId.slice(0, 8)}
                  </div>
                )}
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}
