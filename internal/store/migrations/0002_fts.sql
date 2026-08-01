CREATE VIRTUAL TABLE search_index USING fts5(
  kind UNINDEXED,
  ref_id UNINDEXED,
  title,
  body,
  tokenize = 'porter unicode61'
);

-- Stories.
CREATE TRIGGER stories_search_ai AFTER INSERT ON stories BEGIN
  INSERT INTO search_index (kind, ref_id, title, body)
  VALUES ('story', new.id,
          new.title,
          new.description || ' ' || new.acceptance || ' ' || new.plan || ' ' || new.notes);
END;

CREATE TRIGGER stories_search_au AFTER UPDATE ON stories BEGIN
  DELETE FROM search_index WHERE kind = 'story' AND ref_id = old.id;
  INSERT INTO search_index (kind, ref_id, title, body)
  VALUES ('story', new.id,
          new.title,
          new.description || ' ' || new.acceptance || ' ' || new.plan || ' ' || new.notes);
END;

CREATE TRIGGER stories_search_ad AFTER DELETE ON stories BEGIN
  DELETE FROM search_index WHERE kind = 'story' AND ref_id = old.id;
END;

-- Wiki pages.
CREATE TRIGGER wiki_search_ai AFTER INSERT ON wiki_pages BEGIN
  INSERT INTO search_index (kind, ref_id, title, body)
  VALUES ('wiki', new.id, new.title, new.summary || ' ' || new.body);
END;

CREATE TRIGGER wiki_search_au AFTER UPDATE ON wiki_pages BEGIN
  DELETE FROM search_index WHERE kind = 'wiki' AND ref_id = old.id;
  INSERT INTO search_index (kind, ref_id, title, body)
  VALUES ('wiki', new.id, new.title, new.summary || ' ' || new.body);
END;

CREATE TRIGGER wiki_search_ad AFTER DELETE ON wiki_pages BEGIN
  DELETE FROM search_index WHERE kind = 'wiki' AND ref_id = old.id;
END;

-- Sources.
CREATE TRIGGER sources_search_ai AFTER INSERT ON sources BEGIN
  INSERT INTO search_index (kind, ref_id, title, body)
  VALUES ('source', new.id, new.title, new.body);
END;

CREATE TRIGGER sources_search_ad AFTER DELETE ON sources BEGIN
  DELETE FROM search_index WHERE kind = 'source' AND ref_id = old.id;
END;

-- Backfill rows written before this migration existed.
INSERT INTO search_index (kind, ref_id, title, body)
SELECT 'story', id, title,
       description || ' ' || acceptance || ' ' || plan || ' ' || notes
FROM stories;

INSERT INTO search_index (kind, ref_id, title, body)
SELECT 'wiki', id, title, summary || ' ' || body FROM wiki_pages;

INSERT INTO search_index (kind, ref_id, title, body)
SELECT 'source', id, title, body FROM sources;
