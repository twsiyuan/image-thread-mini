package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"

	_ "github.com/go-sql-driver/mysql"
)

func DbConn(str string) (*sql.DB, error) {
	return sql.Open("mysql", str)
}

func main() {
	dbEndpoint := os.Getenv("DATABASE_ENDPOINT")
	port := os.Getenv("PORT")
	db, err := DbConn(dbEndpoint)
	if err != nil {
		log.Fatal(err)
	}

	// TODO: Not found handleri
	r := mux.NewRouter()
	r.Handle("/", viewStasHandler(db, fileHandler("./index.html"))).Methods(http.MethodGet)
	r.Handle("/export", exportHandler("export.csv", db)).Methods(http.MethodGet)

	r.Handle("/images/{id:[1-9][0-9]*}", imageHandler("id", db)).Methods(http.MethodGet)
	sr := r.PathPrefix("/api").Subrouter()
	sr.Handle("/info", infoHandler(db)).Methods(http.MethodGet)
	sr.Handle("/posts/", postsHandler(db)).Methods(http.MethodGet)
	sr.Handle("/posts/", uploadHandler(db)).Methods(http.MethodPost)

	// TODO: HTTPS, addr from os.env
	if err := http.ListenAndServe(":"+port, r); err != nil {
		fmt.Fprintf(os.Stderr, "Failed, %v\n", err)
	}
}
