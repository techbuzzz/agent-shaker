-- Create daily_standups table
CREATE TABLE IF NOT EXISTS daily_standups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    standup_date DATE NOT NULL,
    did TEXT NOT NULL,
    doing TEXT NOT NULL,
    done TEXT NOT NULL,
    blockers TEXT,
    challenges TEXT,
    references TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_agent_date UNIQUE (agent_id, standup_date)
);

-- Create indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_standups_agent ON daily_standups(agent_id);
CREATE INDEX IF NOT EXISTS idx_standups_project ON daily_standups(project_id);
CREATE INDEX IF NOT EXISTS idx_standups_date ON daily_standups(standup_date);
CREATE INDEX IF NOT EXISTS idx_standups_project_date ON daily_standups(project_id, standup_date);

-- Create agent heartbeats table for tracking agent activity
CREATE TABLE IF NOT EXISTS agent_heartbeats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    heartbeat_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    status VARCHAR(50) DEFAULT 'active',
    metadata JSONB
);

-- Create index for heartbeats
CREATE INDEX IF NOT EXISTS idx_heartbeats_agent ON agent_heartbeats(agent_id);
CREATE INDEX IF NOT EXISTS idx_heartbeats_time ON agent_heartbeats(heartbeat_time);
