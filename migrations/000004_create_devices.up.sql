CREATE TABLE devices (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_id VARCHAR(255) NOT NULL,
    platform VARCHAR(50) NOT NULL,
    name VARCHAR(255),
    sync_cursor TIMESTAMPTZ NOT NULL DEFAULT '1970-01-01 00:00:00+00',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT unique_user_device UNIQUE (user_id, device_id)
);

CREATE INDEX idx_devices_user_id ON devices(user_id);
