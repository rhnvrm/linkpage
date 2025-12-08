package main

import (
	"embed"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	flag "github.com/spf13/pflag"
	"github.com/urfave/negroni"
	_ "modernc.org/sqlite"
)

//go:embed templates/*
var templateFS embed.FS

//go:embed static
var staticFS embed.FS

//go:embed schema.sql config.sample.toml
var setupFS embed.FS

var (
	appMode        = "run_app"
	configFilePath = "config.toml"
)

func init() {
	flag.StringVar(&configFilePath, "config", "config.toml", "path to config file")
	initApp := flag.Bool("init", false, "app initialization, creates a db and config file in current dir")

	flag.Parse()

	if *initApp == true {
		appMode = "init_app"
	}
}

func runApp(configFilePath string) {
	cfg := initConfig(configFilePath)

	// db, err := sqlx.Connect("sqlite", cfg.DBFile)
	// if err != nil {
	// 	log.Fatalln(err)
	// }

	// app := &App{
	// 	Data: Page{
	// 		LogoURL: cfg.PageLogoURL,
	// 		Title:   cfg.PageTitle,
	// 		Intro:   cfg.PageIntro,
	// 		Social:  cfg.Social,
	// 	},
	// 	DB: &LinkDB{db},
	// 	Templates: Templates{
	// 		Home:  newCachedTemplate(template.Must(template.ParseFS(templateFS, "home.html"))),
	// 		Admin: template.Must(template.ParseFS(templateFS, "admin.html")),
	// 	},
	// }

	app, err := NewApp(cfg)
	if err != nil {
		log.Fatal(err)
	}

	// Initial setup of links
	if err := app.UpdateLinks(); err != nil {
		if strings.Contains(err.Error(), "no such table") {
			log.Println("schema not initialized, attempting to initialize schema")

			if err := app.execSchema(); err != nil {
				log.Fatal(err)
			}

			if err := app.UpdateLinks(); err != nil {
				log.Fatal(err)
			}
		} else {
			log.Fatalf("error while trying to update links: %v", err)
		}
	}

	// Run migrations to ensure schema is up to date (after we know table exists)
	if err := app.DB.runMigrations(); err != nil {
		log.Printf("migration warning: %v", err)
	}

	r := mux.NewRouter()
	admin := mux.NewRouter().PathPrefix("/admin").Subrouter().StrictSlash(true)
	r.PathPrefix("/admin").Handler(negroni.New(
		negroni.HandlerFunc(basicAuth(cfg)),
		negroni.Wrap(admin),
	))

	r.HandleFunc("/", app.HandleHome)
	r.HandleFunc("/hits/{id}", app.HandleHits)
	admin.HandleFunc("/", app.HandleAdmin)
	admin.HandleFunc("/links/{id}/weight", app.HandleAdminUpdateWeight)
	admin.HandleFunc("/links/{id}/delete", app.HandleAdminDelete)
	admin.HandleFunc("/links/{id}/update", app.HandleAdminUpdate)
	admin.HandleFunc("/links/new", app.HandleAdminNew)
	r.PathPrefix("/static/app").Handler(customFileServer(cfg.StaticFileDir, staticFS))

	srv := &http.Server{
		Handler:      r,
		Addr:         cfg.HTTPAddr,
		WriteTimeout: cfg.ReadTimeout,
		ReadTimeout:  cfg.WriteTimeout,
	}

	log.Printf("starting server at http://%s", cfg.HTTPAddr)
	log.Fatal(srv.ListenAndServe())
}

func main() {
	runApp(configFilePath)
}
