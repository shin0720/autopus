package memindex

const schemaSQL = `
CREATE TABLE mem_metadata (
	key TEXT PRIMARY KEY,
	value TEXT NOT NULL
);

CREATE TABLE mem_records (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	source_type TEXT NOT NULL,
	source_ref TEXT NOT NULL UNIQUE,
	source_hash TEXT NOT NULL,
	title TEXT NOT NULL,
	summary TEXT NOT NULL,
	tags TEXT NOT NULL,
	spec_id TEXT NOT NULL,
	acceptance_ids TEXT NOT NULL,
	file_refs TEXT NOT NULL,
	package_refs TEXT NOT NULL,
	severity TEXT NOT NULL,
	timestamp TEXT NOT NULL,
	redaction_status TEXT NOT NULL,
	content TEXT NOT NULL,
	source_metadata TEXT NOT NULL
);

CREATE VIRTUAL TABLE mem_records_fts USING fts5(
	title,
	summary,
	tags,
	content,
	content='mem_records',
	content_rowid='id'
);

CREATE TABLE mem_skips (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	path TEXT NOT NULL,
	reason TEXT NOT NULL
);

CREATE INDEX mem_records_type_ref_idx ON mem_records(source_type, source_ref);
CREATE INDEX mem_skips_reason_idx ON mem_skips(reason);
`
