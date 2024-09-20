package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
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

func (l *LinkDB) InsertLink(text, url, imageURL string) error {
	query := `INSERT INTO links (message, url, image_url) VALUES (?, ?, ?);`

	_, err := l.db.Exec(query, text, url, imageURL)
	if err != nil {
		return err
	}

	return nil
}

func (l *LinkDB) UpdateLink(id int, text, url, image string) error {
	query := `UPDATE links SET message=?, url=?, image_url=? WHERE link_id=?;`

	_, err := l.db.Exec(query, text, url, image, id)
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

func initDB(dbFilePath string) {
	file, err := os.Create(dbFilePath)
	if err != nil {
		log.Fatal(err)
	}
	file.Close()

	db, err := sqlx.Connect("sqlite", dbFilePath)
	if err != nil {
		log.Fatal(err)
	}

	if err := execSchema(db); err != nil {
		log.Fatal(err)
	}

	if err := db.Close(); err != nil {
		log.Fatal(err)
	}
}

func execSchema(db *sqlx.DB) error {
	schemaFile, err := setupFS.Open("schema.sql")
	if err != nil {
		return err
	}

	schema, err := ioutil.ReadAll(schemaFile)
	if err != nil {
		return err
	}

	if err := schemaFile.Close(); err != nil {
		return err
	}

	if _, err := db.Exec(string(schema)); err != nil {
		return err
	}

	return nil
}
