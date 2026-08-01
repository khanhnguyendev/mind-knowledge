CREATE TABLE projects (
  id         TEXT PRIMARY KEY,
  name       TEXT NOT NULL UNIQUE,
  repo_path  TEXT NOT NULL,
  git_remote TEXT NOT NULL DEFAULT '',
  status     TEXT NOT NULL DEFAULT 'active',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE epics (
  id          TEXT PRIMARY KEY,
  project_id  TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  title       TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  status      TEXT NOT NULL DEFAULT 'backlog',
  position    INTEGER NOT NULL DEFAULT 0,
  created_at  TEXT NOT NULL,
  updated_at  TEXT NOT NULL
);
CREATE INDEX idx_epics_project ON epics(project_id);

CREATE TABLE stories (
  id          TEXT PRIMARY KEY,
  epic_id     TEXT NOT NULL REFERENCES epics(id) ON DELETE CASCADE,
  title       TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  status      TEXT NOT NULL DEFAULT 'backlog',
  priority    TEXT NOT NULL DEFAULT 'med',
  position    INTEGER NOT NULL DEFAULT 0,
  acceptance  TEXT NOT NULL DEFAULT '',
  plan        TEXT NOT NULL DEFAULT '',
  notes       TEXT NOT NULL DEFAULT '',
  created_at  TEXT NOT NULL,
  updated_at  TEXT NOT NULL
);
CREATE INDEX idx_stories_epic ON stories(epic_id);

CREATE TABLE sources (
  id           TEXT PRIMARY KEY,
  uri          TEXT NOT NULL DEFAULT '',
  title        TEXT NOT NULL,
  kind         TEXT NOT NULL DEFAULT 'note',
  body         TEXT NOT NULL DEFAULT '',
  asset_path   TEXT NOT NULL DEFAULT '',
  content_hash TEXT NOT NULL DEFAULT '',
  ingested_at  TEXT NOT NULL
);
CREATE INDEX idx_sources_hash ON sources(content_hash);

CREATE TABLE wiki_pages (
  id         TEXT PRIMARY KEY,
  slug       TEXT NOT NULL UNIQUE,
  title      TEXT NOT NULL,
  summary    TEXT NOT NULL DEFAULT '',
  kind       TEXT NOT NULL DEFAULT 'concept',
  body       TEXT NOT NULL DEFAULT '',
  status     TEXT NOT NULL DEFAULT 'current',
  project_id TEXT REFERENCES projects(id) ON DELETE SET NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE INDEX idx_wiki_project ON wiki_pages(project_id);

CREATE TABLE links (
  from_kind TEXT NOT NULL,
  from_id   TEXT NOT NULL,
  to_kind   TEXT NOT NULL,
  to_id     TEXT NOT NULL,
  relation  TEXT NOT NULL,
  PRIMARY KEY (from_kind, from_id, to_kind, to_id, relation)
);
CREATE INDEX idx_links_to ON links(to_kind, to_id);

CREATE TABLE tags (
  name TEXT PRIMARY KEY
);

CREATE TABLE entity_tags (
  tag         TEXT NOT NULL REFERENCES tags(name) ON DELETE CASCADE,
  entity_kind TEXT NOT NULL,
  entity_id   TEXT NOT NULL,
  PRIMARY KEY (tag, entity_kind, entity_id)
);
CREATE INDEX idx_entity_tags_entity ON entity_tags(entity_kind, entity_id);

CREATE TABLE log (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  ts         TEXT NOT NULL,
  kind       TEXT NOT NULL,
  project_id TEXT,
  ref        TEXT,
  summary    TEXT NOT NULL
);
CREATE INDEX idx_log_ts ON log(ts);
