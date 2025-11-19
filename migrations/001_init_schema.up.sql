CREATE TABLE IF NOT EXISTS images (
    id UUID PRIMARY KEY,
    status TEXT,
    original_path TEXT,
    created_at TIMESTAMP
)