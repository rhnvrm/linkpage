package main

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
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
	"github.com/otiai10/opengraph/v2"
	flag "github.com/spf13/pflag"
	"github.com/urfave/negroni"
)

//go:embed home.html admin.html
var templateFS embed.FS

//go:embed static
var staticFS embed.FS

//go:embed schema.sql config.sample.toml
var setupFS embed.FS

type Config struct {
	HTTPAddr     string        `koanf:"http_address"`
	ReadTimeout  time.Duration `koanf:"read_timeout"`
	WriteTimeout time.Duration `koanf:"write_timeout"`
	DBFile       string        `koanf:"dbfile"`

	PageLogoURL string `koanf:"page_logo_url"`
	PageTitle   string `koanf:"page_title"`
	PageIntro   string `koanf:"page_intro"`

	Auth CfgAuth `koanf:"auth"`
}

type CfgAuth struct {
	Username string `koanf:"username"`
	Password string `koanf:"password"`
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

	Error   string
	Success string

	OGPURL   string
	OGPImage string
	OGPDesc  string
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

func writeBadRequest(w http.ResponseWriter, message string) {
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte("400 - " + message))
}

func basicAuth(cfg Config) negroni.HandlerFunc {
	return negroni.HandlerFunc(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		user, pass, _ := r.BasicAuth()

		if cfg.Auth.Username != user || cfg.Auth.Password != pass {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized.", http.StatusUnauthorized)
			return
		}

		next(w, r)
	})
}

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
			Home:  newCachedTemplate(template.Must(template.ParseFS(templateFS, "home.html"))),
			Admin: template.Must(template.ParseFS(templateFS, "admin.html")),
		},
	}

	// Initial setup of links
	app.UpdateLinks()

	r := mux.NewRouter()
	admin := mux.NewRouter().PathPrefix("/admin").Subrouter().StrictSlash(true)
	r.PathPrefix("/admin").Handler(negroni.New(
		negroni.HandlerFunc(basicAuth(cfg)),
		negroni.Wrap(admin),
	))

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if err := app.Templates.Home.Write(w); err != nil {
			log.Printf("error while writing template: %v", err)
			writeInternalServerErr(w)
		}
	})

	r.HandleFunc("/hits/{id}", func(w http.ResponseWriter, r *http.Request) {
		rawID, ok := mux.Vars(r)["id"]
		if !ok {
			writeBadRequest(w, "id missing")
			return
		}

		id, err := strconv.Atoi(rawID)
		if err != nil {
			log.Printf("error while getting links: %v", err)
			writeBadRequest(w, "bad id, got "+rawID)
			return
		}

		if err := app.DB.IncrementHit(id); err != nil {
			log.Printf("error while incrementing hits: %v", err)
			writeInternalServerErr(w)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
		return
	})

	renderAdminPage := func(data Page) func(w http.ResponseWriter, r *http.Request) {
		return func(w http.ResponseWriter, r *http.Request) {
			if err := app.Templates.Admin.Execute(w, data); err != nil {
				log.Printf("error while writing template: %v", err)
				writeInternalServerErr(w)
				return
			}
		}
	}

	renderAdminPageWithErrMessage := func(msg string, p Page) func(w http.ResponseWriter, r *http.Request) {
		p.Error = msg
		return renderAdminPage(p)
	}

	admin.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		app.UpdateLinks()
		renderAdminPage(app.Data)(w, r)
	})

	admin.HandleFunc("/links/{id}/weight", func(w http.ResponseWriter, r *http.Request) {
		keys, ok := r.URL.Query()["action"]

		if !ok || len(keys[0]) < 1 {
			renderAdminPageWithErrMessage("action is missing", app.Data)(w, r)
			return
		}

		action := keys[0]

		rawID, ok := mux.Vars(r)["id"]
		if !ok {
			renderAdminPageWithErrMessage("id is missing", app.Data)(w, r)
			return
		}

		id, err := strconv.Atoi(rawID)
		if err != nil {
			renderAdminPageWithErrMessage("bad id, got: "+rawID, app.Data)(w, r)
			return
		}

		if err := app.DB.UpdateWeight(id, action); err != nil {
			renderAdminPageWithErrMessage(
				fmt.Sprintf("error while updating link: %v", err),
				app.Data)(w, r)
			return
		}

		if err := app.UpdateLinks(); err != nil {
			renderAdminPageWithErrMessage(
				fmt.Sprintf("error while updating link: %v", err),
				app.Data)(w, r)
			return
		}

		renderAdminPage(app.Data)(w, r)
	})

	admin.HandleFunc("/links/{id}/delete", func(w http.ResponseWriter, r *http.Request) {
		rawID, ok := mux.Vars(r)["id"]
		if !ok {
			renderAdminPageWithErrMessage("id is missing", app.Data)(w, r)
			return
		}

		id, err := strconv.Atoi(rawID)
		if err != nil {
			renderAdminPageWithErrMessage("bad id, got: "+rawID, app.Data)(w, r)
			return
		}

		if err := app.DB.DeleteLink(id); err != nil {
			renderAdminPageWithErrMessage(
				fmt.Sprintf("error while deleting link: %v", err),
				app.Data)(w, r)
			return
		}

		if err := app.UpdateLinks(); err != nil {
			renderAdminPageWithErrMessage(
				fmt.Sprintf("error while updating links: %v", err),
				app.Data)(w, r)
		}

		renderAdminPage(app.Data)(w, r)
	})

	admin.HandleFunc("/links/{id}/update", func(w http.ResponseWriter, r *http.Request) {
		rawID, ok := mux.Vars(r)["id"]
		if !ok {
			renderAdminPageWithErrMessage("id is missing", app.Data)(w, r)
			return
		}

		id, err := strconv.Atoi(rawID)
		if err != nil {
			renderAdminPageWithErrMessage("bad id, got: "+rawID, app.Data)(w, r)
			return
		}

		r.ParseForm()

		text := r.Form.Get("text")
		url := r.Form.Get("url")
		imageURL := r.Form.Get("image_url")

		if url == "" {
			renderAdminPageWithErrMessage("url is missing", app.Data)(w, r)
			return
		}
		if text == "" {
			renderAdminPageWithErrMessage("text is missing", app.Data)(w, r)
			return
		}
		if imageURL == "" {
			renderAdminPageWithErrMessage("image_url is missing", app.Data)(w, r)
			return
		}

		if err := app.DB.UpdateLink(id, text, url, imageURL); err != nil {
			renderAdminPageWithErrMessage(
				fmt.Sprintf("error while updating link: %v", err),
				app.Data)(w, r)
			return
		}

		if err := app.UpdateLinks(); err != nil {
			renderAdminPageWithErrMessage(
				fmt.Sprintf("error while updating links: %v", err),
				app.Data)(w, r)
			return
		}

		renderAdminPage(app.Data)(w, r)
	})

	admin.HandleFunc("/links/new", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()

		text := r.Form.Get("text")
		url := r.Form.Get("url")
		imageURL := r.Form.Get("image_url")
		submitType := r.Form.Get("submit")

		if url == "" {
			renderAdminPageWithErrMessage("url is missing", app.Data)(w, r)
		}

		if submitType == "Fetch Data" {
			ogp, err := opengraph.Fetch(url)
			if err != nil {
				renderAdminPageWithErrMessage(
					fmt.Sprintf("error while fetching link: %v", err),
					app.Data)(w, r)
				return
			}

			ogp.ToAbs()
			if len(ogp.Image) > 0 {
				imageURL = ogp.Image[0].URL
			}

			p := app.Data
			p.OGPImage = imageURL
			p.OGPDesc = ogp.Title
			p.OGPURL = url
			renderAdminPage(p)(w, r)
			return
		}

		if text == "" {
			renderAdminPageWithErrMessage("text is missing", app.Data)(w, r)
		}

		if imageURL == "" {
			ogp, err := opengraph.Fetch(url)
			if err != nil {
				renderAdminPageWithErrMessage(
					fmt.Sprintf("error while fetching link: %v", err),
					app.Data)(w, r)
				return
			}

			ogp.ToAbs()
			if len(ogp.Image) > 0 {
				imageURL = ogp.Image[0].URL
			}
		}

		if err := app.DB.InsertLink(text, url, imageURL); err != nil {
			renderAdminPageWithErrMessage(
				fmt.Sprintf("error while inserting link: %v", err),
				app.Data)(w, r)
			return
		}

		if err := app.UpdateLinks(); err != nil {
			log.Printf("error while updating links: %v", err)
			writeInternalServerErr(w)
			return
		}

		p := app.Data
		p.Success = "New link inserted!"
		renderAdminPage(p)(w, r)
	})

	r.PathPrefix("/static/").Handler(http.FileServer(http.FS(staticFS)))

	srv := &http.Server{
		Handler:      r,
		Addr:         cfg.HTTPAddr,
		WriteTimeout: cfg.ReadTimeout,
		ReadTimeout:  cfg.WriteTimeout,
	}

	log.Println("starting server at", cfg.HTTPAddr)
	log.Fatal(srv.ListenAndServe())
}

func initApp() {
	file, err := os.Create("app.db")
	if err != nil {
		log.Fatal(err)
	}
	file.Close()

	db, err := sqlx.Connect("sqlite3", "app.db")
	if err != nil {
		log.Fatal(err)
	}

	schemaFile, err := setupFS.Open("schema.sql")
	if err != nil {
		log.Fatal(err)
	}

	schema, err := ioutil.ReadAll(schemaFile)
	if err != nil {
		log.Fatal(err)
	}

	if err := schemaFile.Close(); err != nil {
		log.Fatal(err)
	}

	if _, err := db.Exec(string(schema)); err != nil {
		log.Fatal(err)
	}

	if err := db.Close(); err != nil {
		log.Fatal(err)
	}

	outCfgFile, err := os.Create("config.toml")
	if err != nil {
		log.Fatal(err)
	}
	defer outCfgFile.Close()

	setupCfgFile, err := setupFS.Open("config.sample.toml")
	if err != nil {
		log.Fatal(err)
	}
	defer setupCfgFile.Close()

	if _, err := io.Copy(outCfgFile, setupCfgFile); err != nil {
		log.Fatal(err)
	}

	log.Println("config.toml and app.db generated.")
}

func main() {
	log.Println(appMode)
	switch appMode {
	case "init_app":
		initApp()
	case "run_app":
		runApp(configFilePath)
	default:
		runApp(configFilePath)
	}
}
