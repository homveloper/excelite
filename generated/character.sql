CREATE TABLE character (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME,
    updated_at DATETIME,
    deleted_at DATETIME,
    Attack REAL,
    Class TEXT,
    Defense REAL,
    Hp REAL,
    "Index" TEXT,
    Level INTEGER,
    Mp REAL,
    Name TEXT,
    Skills TEXT,
    Speed REAL,
    Type TEXT
);

CREATE INDEX IF NOT EXISTS idx_character_name ON character (Name);