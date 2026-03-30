-- Create metrics table for storing gauge and counter values
CREATE TABLE IF NOT EXISTS metrics (
    id VARCHAR(255) PRIMARY KEY,
    type VARCHAR(50) NOT NULL CHECK (type IN ('gauge', 'counter')),
    value DOUBLE PRECISION,
    delta BIGINT
);

-- Create index for faster type-based queries
CREATE INDEX IF NOT EXISTS idx_metrics_type ON metrics(type);
