PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS sessions (
  id TEXT PRIMARY KEY,
  code TEXT NOT NULL UNIQUE,
  create_op_id TEXT NOT NULL UNIQUE,
  state TEXT NOT NULL CHECK (state IN ('collecting', 'locked', 'completed', 'abandoned')),
  revision INTEGER NOT NULL DEFAULT 1 CHECK (revision >= 1),
  catalog_revision_id INTEGER NOT NULL REFERENCES catalog_revisions(id),
  target_sha256 TEXT NOT NULL,
  pos_a TEXT NOT NULL,
  pos_b TEXT NOT NULL,
  pos_c TEXT NOT NULL,
  pos_d TEXT NOT NULL,
  protocol_json TEXT NOT NULL,
  help_version TEXT NOT NULL,
  record_json TEXT NOT NULL,
  locked_record_json TEXT,
  collection_ms INTEGER NOT NULL DEFAULT 0 CHECK (collection_ms >= 0),
  choice_ms INTEGER NOT NULL DEFAULT 0 CHECK (choice_ms >= 0),
  timing_seq INTEGER NOT NULL DEFAULT 0 CHECK (timing_seq >= 0),
  paused INTEGER NOT NULL DEFAULT 0 CHECK (paused IN (0, 1)),
  step INTEGER NOT NULL DEFAULT 1 CHECK (step >= 1 AND step <= 6),
  tentative_choice TEXT CHECK (tentative_choice IS NULL OR tentative_choice IN ('A', 'B', 'C', 'D')),
  tentative_confidence INTEGER CHECK (tentative_confidence IS NULL OR (tentative_confidence BETWEEN 0 AND 100)),
  confirmed_choice TEXT CHECK (confirmed_choice IS NULL OR confirmed_choice IN ('A', 'B', 'C', 'D')),
  confirmed_confidence INTEGER CHECK (confirmed_confidence IS NULL OR (confirmed_confidence BETWEEN 0 AND 100)),
  hit INTEGER CHECK (hit IS NULL OR hit IN (0, 1)),
  comment TEXT NOT NULL DEFAULT '',
  comment_updated_at TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  locked_at TEXT,
  completed_at TEXT,
  abandoned_at TEXT,
  abandoned_after_alts INTEGER NOT NULL DEFAULT 0 CHECK (abandoned_after_alts IN (0, 1)),
  alts_authorized_at TEXT,
  lease_token TEXT,
  lease_until TEXT,
  opened_examples_json TEXT NOT NULL DEFAULT '[]',
  CHECK (
    (state = 'collecting' AND confirmed_choice IS NULL AND hit IS NULL)
    OR (state = 'locked' AND confirmed_choice IS NULL AND hit IS NULL AND locked_record_json IS NOT NULL)
    OR (state = 'completed' AND confirmed_choice IS NOT NULL AND hit IS NOT NULL AND locked_record_json IS NOT NULL)
    OR (state = 'abandoned' AND confirmed_choice IS NULL AND hit IS NULL)
  )
);

CREATE UNIQUE INDEX IF NOT EXISTS sessions_one_nonterminal
  ON sessions((1))
  WHERE state IN ('collecting', 'locked');

CREATE TABLE IF NOT EXISTS session_events (
  id INTEGER PRIMARY KEY,
  session_id TEXT NOT NULL REFERENCES sessions(id),
  at TEXT NOT NULL,
  kind TEXT NOT NULL,
  payload_json TEXT NOT NULL DEFAULT '{}'
);
