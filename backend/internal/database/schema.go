package database

import "strings"

// The tables. They match the live Neon database exactly (names, types,
// indexes), so it keeps working untouched. "IF NOT EXISTS" makes setup
// idempotent: it only adds what is missing.
// Times are stored in UTC and sent to clients as Unix seconds.
//
// Column changes to existing tables need a real migration, not an edit here.
const schemaTemplate = `
CREATE TABLE IF NOT EXISTS users (
	id {{serial}},
	google_sub VARCHAR,
	email VARCHAR(320) NOT NULL,
	name VARCHAR(120) NOT NULL,
	picture_url VARCHAR(1000),
	role VARCHAR(20),
	created_at {{timestamp}} NOT NULL,
	PRIMARY KEY (id)
);
CREATE UNIQUE INDEX IF NOT EXISTS ix_users_email ON users (email);
CREATE UNIQUE INDEX IF NOT EXISTS ix_users_google_sub ON users (google_sub);

CREATE TABLE IF NOT EXISTS applications (
	id {{serial}},
	user_id INTEGER NOT NULL,
	role VARCHAR(20) NOT NULL,
	full_name VARCHAR(120) NOT NULL,
	phone VARCHAR(20) NOT NULL,
	citizenship_number VARCHAR(40) NOT NULL,
	citizenship_district VARCHAR(60) NOT NULL,
	council_number VARCHAR(40),
	institution VARCHAR(160),
	recommender_name VARCHAR(120),
	recommender_nmc VARCHAR(40),
	status VARCHAR(20) NOT NULL,
	rejection_reason VARCHAR(500),
	submitted_at {{timestamp}} NOT NULL,
	reviewed_at {{timestamp}},
	reviewed_by VARCHAR(320),
	PRIMARY KEY (id),
	FOREIGN KEY(user_id) REFERENCES users (id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS ix_applications_status ON applications (status);
CREATE UNIQUE INDEX IF NOT EXISTS ix_applications_user_id ON applications (user_id);

CREATE TABLE IF NOT EXISTS chats (
	id VARCHAR(32) NOT NULL,
	user_id INTEGER NOT NULL,
	title VARCHAR(80) NOT NULL,
	created_at {{timestamp}} NOT NULL,
	updated_at {{timestamp}} NOT NULL,
	PRIMARY KEY (id),
	FOREIGN KEY(user_id) REFERENCES users (id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS ix_chats_updated_at ON chats (updated_at);
CREATE INDEX IF NOT EXISTS ix_chats_user_id ON chats (user_id);

CREATE TABLE IF NOT EXISTS help_records (
	id {{serial}},
	professional_id INTEGER NOT NULL,
	consultation_id VARCHAR(32) NOT NULL,
	patient_name VARCHAR(60) NOT NULL,
	started_at {{timestamp}} NOT NULL,
	ended_at {{timestamp}} NOT NULL,
	duration_seconds INTEGER NOT NULL,
	PRIMARY KEY (id),
	FOREIGN KEY(professional_id) REFERENCES users (id) ON DELETE CASCADE,
	UNIQUE (consultation_id)
);
CREATE INDEX IF NOT EXISTS ix_help_records_professional_id ON help_records (professional_id);

CREATE TABLE IF NOT EXISTS sessions (
	token_hash VARCHAR(64) NOT NULL,
	user_id INTEGER NOT NULL,
	created_at {{timestamp}} NOT NULL,
	expires_at {{timestamp}} NOT NULL,
	PRIMARY KEY (token_hash),
	FOREIGN KEY(user_id) REFERENCES users (id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS ix_sessions_user_id ON sessions (user_id);

CREATE TABLE IF NOT EXISTS audit_events (
	id {{serial}},
	application_id INTEGER NOT NULL,
	actor_email VARCHAR(320) NOT NULL,
	action VARCHAR(20) NOT NULL,
	reason VARCHAR(500),
	at {{timestamp}} NOT NULL,
	PRIMARY KEY (id),
	FOREIGN KEY(application_id) REFERENCES applications (id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS ix_audit_events_application_id ON audit_events (application_id);

CREATE TABLE IF NOT EXISTS chat_messages (
	id {{serial}},
	chat_id VARCHAR NOT NULL,
	role VARCHAR(10) NOT NULL,
	content TEXT NOT NULL,
	created_at {{timestamp}} NOT NULL,
	PRIMARY KEY (id),
	FOREIGN KEY(chat_id) REFERENCES chats (id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS ix_chat_messages_chat_id ON chat_messages (chat_id);

CREATE TABLE IF NOT EXISTS documents (
	id {{serial}},
	application_id INTEGER NOT NULL,
	kind VARCHAR(40) NOT NULL,
	content_type VARCHAR(60) NOT NULL,
	size INTEGER NOT NULL,
	data {{bytes}} NOT NULL,
	created_at {{timestamp}} NOT NULL,
	PRIMARY KEY (id),
	FOREIGN KEY(application_id) REFERENCES applications (id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS ix_documents_application_id ON documents (application_id);
`

// schema returns the setup statements for a database kind, one per entry.
// In SQLite an "INTEGER" primary key is the auto-numbered row id.
func schema(kind string) []string {
	types := strings.NewReplacer(
		"{{serial}}", "SERIAL NOT NULL",
		"{{timestamp}}", "TIMESTAMP WITH TIME ZONE",
		"{{bytes}}", "BYTEA",
	)
	if kind == "sqlite" {
		types = strings.NewReplacer(
			"{{serial}}", "INTEGER NOT NULL",
			"{{timestamp}}", "DATETIME",
			"{{bytes}}", "BLOB",
		)
	}
	var statements []string
	for _, statement := range strings.Split(types.Replace(schemaTemplate), ";") {
		if statement = strings.TrimSpace(statement); statement != "" {
			statements = append(statements, statement)
		}
	}
	return statements
}
