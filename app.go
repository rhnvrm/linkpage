package main

import (
	"fmt"
	"sync"
	"text/template"

	"github.com/jmoiron/sqlx"
)

type App struct {
	Data      Page
	DB        *LinkDB
	Templates Templates
	sync.RWMutex
}

func NewApp(cfg Config) (*App, error) {
	db, err := newDB(cfg.DBFile)
	if err != nil {
		return nil, err
	}

	app := &App{
		Data: Page{
			LogoURL: cfg.PageLogoURL,
			Title:   cfg.PageTitle,
			Intro:   cfg.PageIntro,
			Social:  cfg.Social,
		},
		DB: &LinkDB{db},
		Templates: Templates{
			Home: newCachedTemplate(
				template.Must(template.ParseFS(templateFS, "templates/home.html", "templates/utils.html"))),
			Admin: template.Must(template.ParseFS(templateFS, "templates/admin.html", "templates/utils.html")),
		},
	}

	return app, nil
}

func (app *App) UpdateLinks() error {
	links, err := app.DB.GetLinks()
	if err != nil {
		return fmt.Errorf("error while getting links: %v", err)
	}

	app.Data.Links = links

	if err := app.Templates.Home.Save(app.Data); err != nil {
		return fmt.Errorf("failed to save template: %v", err)
	}

	return nil
}

func newDB(path string) (*sqlx.DB, error) {
	db, err := sqlx.Connect("sqlite", path)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func newCachedTemplate(tmpl *template.Template) *cachedTemplate {
	return &cachedTemplate{
		Template: tmpl,
		rawData:  nil,
	}
}
