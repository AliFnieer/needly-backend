-- Create households table
CREATE TABLE IF NOT EXISTS households (
    id         BIGSERIAL PRIMARY KEY,
    name       VARCHAR(100) NOT NULL,
    owner_id   BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create household_members table
CREATE TABLE IF NOT EXISTS household_members (
    id            BIGSERIAL PRIMARY KEY,
    household_id  BIGINT NOT NULL REFERENCES households(id) ON DELETE CASCADE,
    user_id       BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role          VARCHAR(20) NOT NULL DEFAULT 'member',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (household_id, user_id)
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_household_members_household_id ON household_members (household_id);
CREATE INDEX IF NOT EXISTS idx_household_members_user_id ON household_members (user_id);