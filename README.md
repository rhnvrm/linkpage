# linkpage

LinkPage is a FOSS Selfhosted alternative to link listing websites.

## Demo

![demo](static/demo.png)
![demo_admin](static/demo_admin.png)

## Developer Setup

0. `git clone https://github.com/rhnvrm/linkpage.git`

1. Initialize SQL schema from `schema.sql` by copying the schema using sqlite:

```
sqlite3 app.db

sqlite> (paste and run schema)
```

2. Edit `config.toml`

3. Run the app

`go run main.go`

4. Insert new entries under `/admin` page.
