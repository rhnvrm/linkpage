package main

import (
	"path/filepath"
	"testing"
)

func newTestLinkDB(t *testing.T, schema string) *LinkDB {
	t.Helper()

	db, err := newDB(filepath.Join(t.TempDir(), "linkpage.db"))
	if err != nil {
		t.Fatalf("new db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	return &LinkDB{db: db}
}

func TestReplaceLinksClearsDatabaseWhenSeedIsEmpty(t *testing.T) {
	linkDB := newTestLinkDB(t, `
		CREATE TABLE links (
			link_id INTEGER PRIMARY KEY,
			url TEXT NOT NULL,
			message TEXT NOT NULL,
			description TEXT DEFAULT '' NOT NULL,
			image_url TEXT NOT NULL,
			weight INTEGER DEFAULT 0 NOT NULL,
			hits INTEGER DEFAULT 0 NOT NULL
		);
	`)

	if err := linkDB.InsertLink("Existing", "already there", "https://existing.example", ""); err != nil {
		t.Fatalf("insert existing link: %v", err)
	}

	if err := linkDB.ReplaceLinks([]SeedLink{}); err != nil {
		t.Fatalf("replace links with empty seed: %v", err)
	}

	links, err := linkDB.GetLinks()
	if err != nil {
		t.Fatalf("get links: %v", err)
	}

	if len(links) != 0 {
		t.Fatalf("expected empty link set after replacement, got %d link(s)", len(links))
	}
}

func TestReplaceLinksRollsBackOnInsertFailure(t *testing.T) {
	linkDB := newTestLinkDB(t, `
		CREATE TABLE links (
			link_id INTEGER PRIMARY KEY,
			url TEXT NOT NULL,
			message TEXT NOT NULL,
			description TEXT DEFAULT '' NOT NULL,
			image_url TEXT NOT NULL,
			weight INTEGER DEFAULT 0 NOT NULL CHECK(weight >= 0),
			hits INTEGER DEFAULT 0 NOT NULL
		);
	`)

	if err := linkDB.InsertLink("Existing", "already there", "https://existing.example", ""); err != nil {
		t.Fatalf("insert existing link: %v", err)
	}

	err := linkDB.ReplaceLinks([]SeedLink{
		{
			URL:         "https://new.example",
			Message:     "New Link",
			Description: "valid replacement",
			ImageURL:    "",
			Weight:      10,
		},
		{
			URL:         "https://broken.example",
			Message:     "Broken Link",
			Description: "fails check constraint",
			ImageURL:    "",
			Weight:      -1,
		},
	})
	if err == nil {
		t.Fatal("expected replace links to fail")
	}

	links, err := linkDB.GetLinks()
	if err != nil {
		t.Fatalf("get links: %v", err)
	}

	if len(links) != 1 {
		t.Fatalf("expected original link set to remain after rollback, got %d link(s)", len(links))
	}

	if links[0].Text != "Existing" || links[0].URL != "https://existing.example" {
		t.Fatalf("expected original link to survive rollback, got %#v", links[0])
	}
}
