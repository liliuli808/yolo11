CREATE TABLE IF NOT EXISTS posts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    persona_id UUID NOT NULL REFERENCES personas(id) ON DELETE CASCADE,
    topic_id UUID NOT NULL REFERENCES topics(id) ON DELETE RESTRICT,
    content TEXT NOT NULL CHECK (length(content) <= 2000),
    moderation_state VARCHAR(16) NOT NULL DEFAULT 'pendingReview'
        CHECK (moderation_state IN ('published', 'pendingReview', 'rejected', 'hidden', 'deleted')),
    reaction_counts JSONB NOT NULL DEFAULT '{}',
    reply_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_posts_persona ON posts(persona_id);
CREATE INDEX idx_posts_topic ON posts(topic_id);
CREATE INDEX idx_posts_created_at ON posts(created_at DESC);

CREATE TABLE IF NOT EXISTS comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    persona_id UUID NOT NULL REFERENCES personas(id) ON DELETE CASCADE,
    content TEXT NOT NULL CHECK (length(content) <= 2000),
    moderation_state VARCHAR(16) NOT NULL DEFAULT 'pendingReview'
        CHECK (moderation_state IN ('published', 'pendingReview', 'rejected', 'hidden', 'deleted')),
    reaction_counts JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_comments_post ON comments(post_id);
CREATE INDEX idx_comments_persona ON comments(persona_id);
CREATE INDEX idx_comments_created_at ON comments(created_at DESC);

CREATE TABLE IF NOT EXISTS reactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    target_type VARCHAR(16) NOT NULL CHECK (target_type IN ('post', 'comment')),
    target_id UUID NOT NULL,
    persona_id UUID NOT NULL REFERENCES personas(id) ON DELETE CASCADE,
    type VARCHAR(16) NOT NULL CHECK (type IN ('like')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (target_type, target_id, persona_id)
);

CREATE INDEX idx_reactions_target ON reactions(target_type, target_id);

CREATE TABLE IF NOT EXISTS media_assets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    persona_id UUID NOT NULL REFERENCES personas(id) ON DELETE CASCADE,
    url VARCHAR(1024) NOT NULL,
    mime_type VARCHAR(128) NOT NULL,
    width INTEGER NOT NULL,
    height INTEGER NOT NULL,
    thumbnail_url VARCHAR(1024),
    status VARCHAR(16) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'ready', 'rejected')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
