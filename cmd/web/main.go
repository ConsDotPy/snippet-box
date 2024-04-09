package main

import (
	"database/sql"
	"flag"
	"log"
	"net/http"
	"os"
	"text/template"

	"snippetbox.consdotpy.xyz/internal/models"

	_ "github.com/go-sql-driver/mysql"
)

type configuration struct {
	addr      string
	staticDir string
	dsn       string
}

type application struct {
	errorLog      *log.Logger
	infoLog       *log.Logger
	config        configuration
	snippets      *models.SnippetModel
	templateCache map[string]*template.Template
}

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}

// main is the application's entry point.
func main() {
	var config configuration
	flag.StringVar(&config.addr, "addr", ":4000", "HTTP network address")
	flag.StringVar(&config.staticDir, "static-dir", "./ui/static/", "Path to static assets")
	flag.StringVar(&config.dsn, "dsn", "", "MySQL data source name")
	flag.Parse()

	infoLog := log.New(
		os.Stdout,
		"INFO\t",
		log.Ldate|log.Ltime|log.LUTC,
	)
	errorLog := log.New(
		os.Stderr,
		"ERROR\t",
		log.Ldate|log.Ltime|log.LUTC|log.Llongfile,
	)

	db, err := openDB(config.dsn)
	if err != nil {
		errorLog.Fatal(err)
	}

	defer db.Close()

	snippets, err := models.NewSnippetModel(db)
	if err != nil {
		errorLog.Fatal(err)
	}

	defer snippets.InsertStmt.Close()
	defer snippets.GetStmt.Close()
	defer snippets.LatestStmt.Close()

	templateCache, err := newTemplateCache()
	if err != nil {
		errorLog.Fatal(err)
	}

	app := &application{
		errorLog:      errorLog,
		infoLog:       infoLog,
		config:        config,
		snippets:      snippets,
		templateCache: templateCache,
	}

	srv := &http.Server{
		Addr:     config.addr,
		ErrorLog: errorLog,
		Handler:  app.routes(),
	}

	// Start server.
	infoLog.Printf("Starting server on %s", config.addr)
	err = srv.ListenAndServe()

	// Log and exit on server start error.
	errorLog.Fatal(err)
}
