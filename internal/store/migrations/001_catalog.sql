PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS schema_migrations (
  version INTEGER PRIMARY KEY,
  applied_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS preferences (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS catalog_revisions (
  id INTEGER PRIMARY KEY,
  source_label TEXT NOT NULL,
  activated_at TEXT NOT NULL,
  eligible_count INTEGER NOT NULL,
  excluded_count INTEGER NOT NULL,
  report_json TEXT NOT NULL,
  active INTEGER NOT NULL DEFAULT 0 CHECK (active IN (0, 1))
);

CREATE UNIQUE INDEX IF NOT EXISTS catalog_revisions_one_active
  ON catalog_revisions(active) WHERE active = 1;

CREATE TABLE IF NOT EXISTS catalog_images (
  revision_id INTEGER NOT NULL REFERENCES catalog_revisions(id),
  sha256 TEXT NOT NULL,
  original_relpath TEXT NOT NULL,
  display_relpath TEXT NOT NULL,
  width INTEGER NOT NULL,
  height INTEGER NOT NULL,
  bytes INTEGER NOT NULL,
  primary_source_id TEXT,
  sources_json TEXT NOT NULL,
  PRIMARY KEY (revision_id, sha256)
);

CREATE TABLE IF NOT EXISTS catalog_exclusions (
  revision_id INTEGER NOT NULL REFERENCES catalog_revisions(id),
  sha256 TEXT,
  path TEXT,
  reason TEXT NOT NULL
);
