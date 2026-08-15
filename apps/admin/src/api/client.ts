import type { Session, ModerationCase, ModerationAction, CursorPage, ApiError } from '../types';

const API_BASE = '/v1';

function getToken(): string | null {
  return localStorage.getItem('lantern_admin_token');
}

function setToken(token: string) {
  localStorage.setItem('lantern_admin_token', token);
}

export function clearSession() {
  localStorage.removeItem('lantern_admin_token');
  localStorage.removeItem('lantern_admin_refresh');
}

export function isAuthenticated(): boolean {
  return !!getToken();
}

async function request<T>(method: string, path: string, body?: unknown, idempotencyKey?: string): Promise<T> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  };
  const token = getToken();
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }
  if (idempotencyKey) {
    headers['Idempotency-Key'] = idempotencyKey;
  }

  const response = await fetch(`${API_BASE}${path}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  });

  if (response.status === 204) {
    return undefined as T;
  }

  const data = await response.json().catch(() => ({}));
  if (!response.ok) {
    const err = data as ApiError;
    const message = err?.message || `Request failed with status ${response.status}`;
    throw new Error(message);
  }
  return data as T;
}

export async function sendEmailCode(email: string): Promise<void> {
  return request('POST', '/auth/email-codes', { email, purpose: 'login' }, `admin-code-${Date.now()}`);
}

export async function createEmailSession(email: string, code: string): Promise<Session> {
  const session = await request<Session>('POST', '/auth/email-sessions', { email, code }, `admin-session-${Date.now()}`);
  setToken(session.accessToken);
  localStorage.setItem('lantern_admin_refresh', session.refreshToken);
  return session;
}

export async function listCases(status?: string, cursor?: string | null): Promise<CursorPage<ModerationCase>> {
  const params = new URLSearchParams();
  if (status) params.set('status', status);
  if (cursor) params.set('cursor', cursor);
  return request('GET', `/moderation/cases?${params.toString()}`);
}

export async function getCase(id: string): Promise<ModerationCase> {
  return request('GET', `/moderation/cases/${id}`);
}

export async function updateCase(
  id: string,
  status: string,
  outcome?: string,
  notes?: string
): Promise<ModerationCase> {
  return request(
    'PATCH',
    `/moderation/cases/${id}`,
    { status, outcome, notes },
    `admin-case-${id}-${Date.now()}`
  );
}

export async function listCaseActions(id: string): Promise<ModerationAction[]> {
  return request('GET', `/moderation/cases/${id}/actions`);
}

export type Outcome = 'noAction' | 'warn' | 'hide' | 'remove' | 'restore' | 'restrictPersona' | 'suspendAccount' | 'banAccount';
