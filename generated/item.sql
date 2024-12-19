CREATE TABLE item (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME,
    updated_at DATETIME,
    deleted_at DATETIME,
    Effects TEXT,
    "Index" TEXT,
    Level_req INTEGER,
    Name TEXT,
    Price INTEGER,
    Rarity INTEGER,
    Stackable BOOLEAN,
    Type TEXT,
    Weight REAL
);

CREATE INDEX IF NOT EXISTS idx_item_name ON item (Name);