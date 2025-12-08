package main

import (
	"fmt"
	"io"
	"strconv"

	"github.com/jmoiron/sqlx"
)

type LinkDB struct {
	db *sqlx.DB
}

func (l *LinkDB) GetLinks() ([]Link, error) {
	links := []Link{}
	if err := l.db.Select(&links,
		"SELECT * FROM links ORDER BY weight DESC, link_id ASC;"); err != nil {
		return nil, err
	}

	return links, nil
}

func (l *LinkDB) UpdateWeight(id int, action string) error {
	var queryAction string
	switch action {
	case "up":
		queryAction = "+ 1"
	case "down":
		queryAction = "- 1"
	default:
		return fmt.Errorf("unsupported action: %s", action)
	}

	res, err := l.db.Exec("UPDATE links set weight = weight " + queryAction + " where link_id = " + strconv.Itoa(id) + ";")
	if err != nil {
		return err
	}

	if c, _ := res.RowsAffected(); c == 0 {
		return fmt.Errorf("item not found: %d", id)
	}

	return nil
}

func (l *LinkDB) InsertLink(text, description, url, imageURL string) error {
	query := `INSERT INTO links (message, description, url, image_url) VALUES (?, ?, ?, ?);`

	_, err := l.db.Exec(query, text, description, url, imageURL)
	if err != nil {
		return err
	}

	return nil
}

func (l *LinkDB) UpdateLink(id int, text, description, url, image string) error {
	query := `UPDATE links SET message=?, description=?, url=?, image_url=? WHERE link_id=?;`

	_, err := l.db.Exec(query, text, description, url, image, id)
	if err != nil {
		return err
	}

	return nil
}

func (l *LinkDB) DeleteLink(id int) error {
	query := `DELETE FROM links where link_id = ?;`

	_, err := l.db.Exec(query, id)
	if err != nil {
		return err
	}

	return nil
}

func (l *LinkDB) IncrementHit(id int) error {
	_, err := l.db.Exec("UPDATE links set hits = hits + 1 where link_id = " + strconv.Itoa(id) + ";")
	if err != nil {
		return err
	}

	return nil
}

func (app *App) execSchema() error {
	schemaFile, err := setupFS.Open("schema.sql")
	if err != nil {
		return err
	}

	schema, err := io.ReadAll(schemaFile)
	if err != nil {
		return err
	}

	if err := schemaFile.Close(); err != nil {
		return err
	}

	if _, err := app.DB.db.Exec(string(schema)); err != nil {
		return err
	}

	// Seed with example data
	if err := app.DB.seedExampleData(); err != nil {
		return fmt.Errorf("failed to seed example data: %v", err)
	}

	return nil
}

// seedExampleData inserts example links for new installations
func (l *LinkDB) seedExampleData() error {
	exampleLinks := []struct {
		text        string
		description string
		url         string
		imageURL    string
	}{
		{
			text:        "Getting Started with LinkPage",
			description: "Learn how to customize your LinkPage and add your own links through the admin panel",
			url:         "https://github.com/rhnvrm/linkpage",
			imageURL:    "",
		},
		{
			text:        "View Documentation",
			description: "Comprehensive documentation covering installation, configuration, and deployment options",
			url:         "https://github.com/rhnvrm/linkpage#features",
			imageURL:    "",
		},
		{
			text:        "Get Started",
			description: "Quick setup guide to get your LinkPage running in minutes",
			url:         "https://github.com/rhnvrm/linkpage#get-started",
			imageURL:    "",
		},
	}

	for _, link := range exampleLinks {
		err := l.InsertLink(link.text, link.description, link.url, link.imageURL)
		if err != nil {
			return err
		}
	}

	return nil
}

// Migration represents a database migration
type Migration struct {
	Version int
	Name    string
	Up      string
}

// migrations contains all database migrations in order
var migrations = []Migration{
	{
		Version: 1,
		Name:    "add_description_column",
		Up:      `ALTER TABLE links ADD COLUMN description TEXT DEFAULT '' NOT NULL;`,
	},
	// Future migrations go here
}

// runMigrations checks and applies necessary schema migrations
func (l *LinkDB) runMigrations() error {
	// Create migrations table if it doesn't exist
	_, err := l.db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %v", err)
	}

	// Get current version
	var currentVersion int
	err = l.db.Get(&currentVersion, `SELECT COALESCE(MAX(version), 0) FROM schema_migrations;`)
	if err != nil {
		return fmt.Errorf("failed to get current migration version: %v", err)
	}

	// Apply pending migrations
	for _, migration := range migrations {
		if migration.Version <= currentVersion {
			continue
		}

		// Check if the migration needs to be applied
		// For version 1 (description column), check if column already exists
		if migration.Version == 1 {
			var columnExists int
			err := l.db.Get(&columnExists, `
				SELECT COUNT(*)
				FROM pragma_table_info('links')
				WHERE name='description';
			`)
			if err != nil {
				return fmt.Errorf("failed to check for description column: %v", err)
			}

			// Skip if column already exists
			if columnExists > 0 {
				// Mark as applied
				_, err := l.db.Exec(`INSERT INTO schema_migrations (version, name) VALUES (?, ?);`,
					migration.Version, migration.Name)
				if err != nil {
					return fmt.Errorf("failed to record migration %d: %v", migration.Version, err)
				}
				continue
			}
		}

		// Execute migration
		_, err := l.db.Exec(migration.Up)
		if err != nil {
			return fmt.Errorf("failed to apply migration %d (%s): %v", migration.Version, migration.Name, err)
		}

		// Record migration
		_, err = l.db.Exec(`INSERT INTO schema_migrations (version, name) VALUES (?, ?);`,
			migration.Version, migration.Name)
		if err != nil {
			return fmt.Errorf("failed to record migration %d: %v", migration.Version, err)
		}
	}

	return nil
}
