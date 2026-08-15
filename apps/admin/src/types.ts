export interface Session {
  accessToken: string;
  refreshToken: string;
  tokenType: string;
  expiresIn: number;
  personaId: string | null;
  isStaff: boolean;
}

export interface ModerationCase {
  id: string;
  targetType: 'post' | 'comment' | 'persona';
  targetId: string;
  reportIds: string[];
  status: 'open' | 'underReview' | 'resolved';
  outcome: string | null;
  notes: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface ModerationAction {
  id: string;
  moderatorUserId: string | null;
  actionType: string;
  targetType: string | null;
  targetId: string | null;
  note: string | null;
  createdAt: string;
}

export interface Report {
  id: string;
  targetType: string;
  targetId: string;
  category: string;
  details: string | null;
  status: string;
  createdAt: string;
  resolvedAt: string | null;
}

export interface ApiError {
  code: string;
  message: string;
  requestId: string;
}

export interface CursorPage<T> {
  data: T[];
  pagination: {
    nextCursor: string | null;
    hasMore: boolean;
    limit: number;
  };
}
