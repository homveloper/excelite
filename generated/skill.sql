CREATE TABLE skill (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME,
    updated_at DATETIME,
    deleted_at DATETIME,
    Cooldown REAL,
    Effects TEXT,
    Element TEXT,
    "Index" TEXT,
    Mp_cost INTEGER,
    Name TEXT,
    Power REAL,
    Requirements TEXT,
    Target_type TEXT
);

CREATE INDEX IF NOT EXISTS idx_skill_name ON skill (Name);