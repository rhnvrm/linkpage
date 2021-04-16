package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
	"text/template"
	"time"

	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/file"
	_ "github.com/mattn/go-sqlite3"
)

type Config struct {
	HTTPAddr     string        `koanf:"http_address"`
	ReadTimeout  time.Duration `koanf:"read_timeout"`
	WriteTimeout time.Duration `koanf:"write_timeout"`
	StaticDir    string        `koanf:"static_dir"`
	DBFile       string        `koanf:"dbfile"`

	PageLogoURL string `koanf:"page_logo_url"`
	PageTitle   string `koanf:"page_title"`
	PageIntro   string `koanf:"page_intro"`
}

type App struct {
	Data      Page
	DB        *LinkDB
	Templates Templates
	sync.RWMutex
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

type Link struct {
	ID       int    `db:"link_id"`
	URL      string `db:"url"`
	Text     string `db:"message"`
	ImageURL string `db:"image_url"`
	Weight   int    `db:"weight"`
	Hits     int    `db:"hits"`
}

type Page struct {
	LogoURL string
	Title   string
	Intro   string
	Links   []Link
}

type cachedTemplate struct {
	*template.Template
	rawData []byte
	sync.RWMutex
}

func newCachedTemplate(tmpl *template.Template) *cachedTemplate {
	return &cachedTemplate{
		Template: tmpl,
		rawData:  nil,
	}
}

func (ct *cachedTemplate) Save(data Page) error {
	var out = bytes.NewBuffer([]byte{})
	if err := ct.Execute(out, data); err != nil {
		return err
	}

	ct.Lock()
	ct.rawData = out.Bytes()
	ct.Unlock()
	return nil
}

func (ct *cachedTemplate) Write(w io.Writer) error {
	ct.RLock()
	defer ct.RUnlock()

	_, err := io.Copy(w, bytes.NewBuffer(ct.rawData))
	if err != nil {
		return err
	}

	return nil
}

type Templates struct {
	Home  *cachedTemplate
	Admin *template.Template
}

type LinkDB struct {
	db *sqlx.DB
}

func (l *LinkDB) GetLinks() ([]Link, error) {
	links := []Link{}
	if err := l.db.Select(&links,
		"SELECT * FROM links ORDER BY weight DESC, link_id DESC;"); err != nil {
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

	_, err := l.db.Exec("UPDATE links set weight = weight " + queryAction + " where link_id = " + strconv.Itoa(id) + ";")
	if err != nil {
		return err
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

func initConfig(configFile string) Config {
	var (
		config Config
		k      = koanf.New(".")
	)

	if err := k.Load(file.Provider(configFile), toml.Parser()); err != nil {
		log.Fatalf("error loading file: %v", err)
	}

	if err := k.Unmarshal("", &config); err != nil {
		log.Fatalf("error while unmarshalling config: %v", err)
	}

	return config
}

func writeInternalServerErr(w http.ResponseWriter) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte("500 - Internal Server Error!"))
}

func main() {
	cfg := initConfig("config.toml")

	db, err := sqlx.Connect("sqlite3", cfg.DBFile)
	if err != nil {
		log.Fatalln(err)
	}

	app := &App{
		Data: Page{
			LogoURL: cfg.PageLogoURL,
			Title:   cfg.PageTitle,
			Intro:   cfg.PageIntro,
		},
		DB: &LinkDB{db},
		Templates: Templates{
			Home:  newCachedTemplate(template.Must(template.ParseFiles("home.html"))),
			Admin: template.Must(template.ParseFiles("admin.html")),
		},
	}

	// Initial setup of links
	app.UpdateLinks()

	r := mux.NewRouter()

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if err := app.Templates.Home.Write(w); err != nil {
			log.Printf("error while writing template: %v", err)
			writeInternalServerErr(w)
		}
	})

	r.HandleFunc("/hits/{id}", func(w http.ResponseWriter, r *http.Request) {
		rawID, ok := mux.Vars(r)["id"]
		if !ok {
			// TODO handle err
			log.Println("id missing")
			writeInternalServerErr(w)
			return
		}

		id, err := strconv.Atoi(rawID)
		if err != nil {
			// TODO handle error
			log.Printf("error while getting links: %v", err)
			writeInternalServerErr(w)
			return
		}

		if err := app.DB.IncrementHit(id); err != nil {
			log.Printf("error while incrementing hits: %v", err)
		}
	})

	r.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
		app.UpdateLinks()
		if err := app.Templates.Admin.Execute(w, app.Data); err != nil {
			log.Printf("error while writing template: %v", err)
			writeInternalServerErr(w)
		}
	})

	r.HandleFunc("/admin/links/{id}/weight", func(w http.ResponseWriter, r *http.Request) {
		keys, ok := r.URL.Query()["action"]

		if !ok || len(keys[0]) < 1 {
			log.Println("Url Param 'action' is missing")
			writeInternalServerErr(w)
			return
		}

		action := keys[0]

		rawID, ok := mux.Vars(r)["id"]
		if !ok {
			// TODO handle err
			log.Println("id missing")
			writeInternalServerErr(w)
			return
		}

		id, err := strconv.Atoi(rawID)
		if err != nil {
			// TODO handle error
			log.Printf("error while getting links: %v", err)
			writeInternalServerErr(w)
			return
		}

		if err := app.DB.UpdateWeight(id, action); err != nil {
			// TODO handle error
			log.Printf("error while getting links: %v", err)
			writeInternalServerErr(w)
			return
		}

		if err := app.UpdateLinks(); err != nil {
			log.Printf("error while updating links: %v", err)
			writeInternalServerErr(w)
			return
		}

		http.Redirect(w, r, "/admin", http.StatusSeeOther)
	})

	r.HandleFunc("/admin/links/{id}/delete", func(w http.ResponseWriter, r *http.Request) {
		rawID, ok := mux.Vars(r)["id"]
		if !ok {
			// TODO handle err
			log.Println("id missing")
			writeInternalServerErr(w)
			return
		}

		id, err := strconv.Atoi(rawID)
		if err != nil {
			// TODO handle error
			log.Printf("error while getting links: %v", err)
			writeInternalServerErr(w)
			return
		}

		if err := app.DB.DeleteLink(id); err != nil {
			// TODO handle error
			log.Printf("error while getting links: %v", err)
			writeInternalServerErr(w)
			return
		}

		if err := app.UpdateLinks(); err != nil {
			log.Printf("error while updating links: %v", err)
			writeInternalServerErr(w)
			return
		}

		http.Redirect(w, r, "/admin", http.StatusSeeOther)
	})

	r.HandleFunc("/admin/links/new", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()

		text := r.Form.Get("text")
		url := r.Form.Get("url")
		imageURL := r.Form.Get("image_url")

		if err := app.DB.InsertLink(text, url, imageURL); err != nil {
			log.Printf("error while updating links: %v", err)
			writeInternalServerErr(w)
			return
		}

		if err := app.UpdateLinks(); err != nil {
			log.Printf("error while updating links: %v", err)
			writeInternalServerErr(w)
			return
		}

		http.Redirect(w, r, "/admin", http.StatusSeeOther)
	})

	r.PathPrefix("/static/").Handler(
		http.StripPrefix("/static/", http.FileServer(http.Dir(cfg.StaticDir))))

	srv := &http.Server{
		Handler:      r,
		Addr:         cfg.HTTPAddr,
		WriteTimeout: cfg.ReadTimeout,
		ReadTimeout:  cfg.WriteTimeout,
	}

	log.Fatal(srv.ListenAndServe())
}
