CREATE TABLE notes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    location GEOGRAPHY(POINT, 4326),
    altitude DOUBLE PRECISION,
    accuracy DOUBLE PRECISION,
    client_id VARCHAR(36),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    CONSTRAINT unique_user_client_id UNIQUE (user_id, client_id)
);

CREATE INDEX idx_notes_user_id ON notes(user_id);
CREATE INDEX idx_notes_user_updated ON notes(user_id, updated_at);
CREATE INDEX idx_notes_location ON notes USING GIST(location);
CREATE INDEX idx_notes_client_id ON notes(client_id) WHERE client_id IS NOT NULL;
CREATE INDEX idx_notes_not_deleted ON notes(user_id) WHERE deleted_at IS NULL;
