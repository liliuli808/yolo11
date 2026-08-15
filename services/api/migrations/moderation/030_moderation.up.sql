CREATE TABLE IF NOT EXISTS blocks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    blocker_persona_id UUID NOT NULL REFERENCES personas(id) ON DELETE CASCADE,
    blocked_persona_id UUID NOT NULL REFERENCES personas(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT blocks_unique_direction UNIQUE (blocker_persona_id, blocked_persona_id),
    CONSTRAINT blocks_no_self_block CHECK (blocker_persona_id != blocked_persona_id)
);

CREATE INDEX IF NOT EXISTS idx_blocks_blocker ON blocks(blocker_persona_id);
CREATE INDEX IF NOT EXISTS idx_blocks_blocked ON blocks(blocked_persona_id);

CREATE TABLE IF NOT EXISTS reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reporter_real_profile_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_type VARCHAR(16) NOT NULL CHECK (target_type IN ('post', 'comment', 'persona')),
    target_id UUID NOT NULL,
    category VARCHAR(32) NOT NULL CHECK (category IN (
        'harassment', 'hateSpeech', 'harmfulContent', 'spam',
        'sexualContent', 'doxxing', 'impersonation', 'illegalContent', 'other'
    )),
    details TEXT CHECK (details IS NULL OR length(details) <= 2000),
    status VARCHAR(16) NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'resolved')),
    moderator_note TEXT CHECK (moderator_note IS NULL OR length(moderator_note) <= 4000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_reports_target ON reports(target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_reports_status ON reports(status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_reports_reporter ON reports(reporter_real_profile_id);

CREATE TABLE IF NOT EXISTS moderation_cases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    target_type VARCHAR(16) NOT NULL CHECK (target_type IN ('post', 'comment', 'persona')),
    target_id UUID NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'underReview', 'resolved')),
    outcome VARCHAR(32) CHECK (outcome IS NULL OR outcome IN (
        'noAction', 'warn', 'hide', 'remove', 'restore',
        'restrictPersona', 'suspendAccount', 'banAccount'
    )),
    notes TEXT CHECK (notes IS NULL OR length(notes) <= 4000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_moderation_cases_status ON moderation_cases(status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_moderation_cases_target ON moderation_cases(target_type, target_id);

CREATE TABLE IF NOT EXISTS case_reports (
    case_id UUID NOT NULL REFERENCES moderation_cases(id) ON DELETE CASCADE,
    report_id UUID NOT NULL REFERENCES reports(id) ON DELETE CASCADE,
    PRIMARY KEY (case_id, report_id)
);

CREATE INDEX IF NOT EXISTS idx_case_reports_report ON case_reports(report_id);

CREATE TABLE IF NOT EXISTS moderation_actions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id UUID REFERENCES moderation_cases(id) ON DELETE SET NULL,
    moderator_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    action_type VARCHAR(32) NOT NULL CHECK (action_type IN (
        'open', 'underReview', 'resolve', 'hide', 'restore', 'warn',
        'restrictPersona', 'suspendAccount', 'banAccount', 'noAction'
    )),
    target_type VARCHAR(16) CHECK (target_type IS NULL OR target_type IN ('post', 'comment', 'persona')),
    target_id UUID,
    note TEXT CHECK (note IS NULL OR length(note) <= 4000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_moderation_actions_case ON moderation_actions(case_id, created_at DESC);
