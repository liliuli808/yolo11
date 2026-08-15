CREATE TABLE IF NOT EXISTS topics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(64) NOT NULL UNIQUE,
    description VARCHAR(256) NOT NULL DEFAULT '',
    category VARCHAR(64) NOT NULL DEFAULT '',
    status VARCHAR(16) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'hidden')),
    slug VARCHAR(64),
    icon_url VARCHAR(512),
    cover_url VARCHAR(512),
    note_count INTEGER NOT NULL DEFAULT 0,
    follower_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS topic_follows (
    persona_id UUID NOT NULL REFERENCES personas(id) ON DELETE CASCADE,
    topic_id UUID NOT NULL REFERENCES topics(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (persona_id, topic_id)
);

CREATE INDEX idx_topics_status ON topics(status);
CREATE INDEX idx_topics_name ON topics(name);
CREATE INDEX idx_topic_follows_topic ON topic_follows(topic_id);

INSERT INTO topics (name, description, category, status, slug)
VALUES
    ('General', 'Open conversation about anything.', 'Everyday', 'active', 'general'),
    ('Reflection', 'Deeper thoughts and personal reflections.', 'Reflection', 'active', 'reflection'),
    ('Creative', 'Share creative work and inspiration.', 'Creative', 'active', 'creative')
ON CONFLICT (name) DO NOTHING;
