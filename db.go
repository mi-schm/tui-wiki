package main

import (
	"database/sql"
	_ "github.com/glebarez/go-sqlite" // Der neue Pure-Go Treiber
)

func initDB() *sql.DB {
	// Der Treibername ist hier "sqlite" statt "sqlite3"
	db, err := sql.Open("sqlite", "wiki.db")
	if err != nil {
		panic(err)
	}

	query := `
	CREATE TABLE IF NOT EXISTS pages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		parent_id INTEGER DEFAULT 0,
		title TEXT NOT NULL,
		content TEXT,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE VIRTUAL TABLE IF NOT EXISTS pages_search USING fts5(title, content, content='pages', content_rowid='id');
	`
	_, err = db.Exec(query)
	if err != nil { panic(err) }

	triggers := `
	CREATE TRIGGER IF NOT EXISTS pages_ai AFTER INSERT ON pages BEGIN
		INSERT INTO pages_search(rowid, title, content) VALUES (new.id, new.title, new.content);
	END;
	CREATE TRIGGER IF NOT EXISTS pages_ad AFTER DELETE ON pages BEGIN
		INSERT INTO pages_search(pages_search, rowid, title, content) VALUES('delete', old.id, old.title, old.content);
	END;
	CREATE TRIGGER IF NOT EXISTS pages_au AFTER UPDATE ON pages BEGIN
		INSERT INTO pages_search(pages_search, rowid, title, content) VALUES('delete', old.id, old.title, old.content);
		INSERT INTO pages_search(rowid, title, content) VALUES (new.id, new.title, new.content);
		UPDATE pages SET updated_at = CURRENT_TIMESTAMP WHERE id = old.id;
	END;
	`
	db.Exec(triggers)
	db.Exec("INSERT INTO pages_search(pages_search) VALUES('rebuild');")
	
	return db
}