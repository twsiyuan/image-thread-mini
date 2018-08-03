package main

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"image/jpeg"
	"image/png"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/unrolled/render"
)

var (
	ren = render.New()
)

type responseError struct {
	Error string
}

func recoveryHandler(outputErr bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				stack := make([]byte, 2048)
				stack = stack[:runtime.Stack(stack, false)]
				displayErr := "Internal server error"
				if outputErr {
					displayErr = fmt.Sprintf("Unexpected error: %v, in %s", err, stack)
				}

				log.SetOutput(os.Stderr)
				log.Printf("Unexpected error: %v, in %s\n", err, stack)

				ren.JSON(w, http.StatusInternalServerError, responseError{displayErr})
			}
		}()

		next.ServeHTTP(w, req)
	})
}

func infoHandler(db *sql.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		numViews := int64(0)
		numPosts := int64(0)

		// TODO: using DbModel, or Merge a single query
		if err := db.QueryRow("SELECT COUNT(*) AS `posts` FROM `posts`").Scan(&numPosts); err != nil {
			panic(err)
		}

		if err := db.QueryRow("SELECT `view` FROM `stats`").Scan(&numViews); err != nil && err != sql.ErrNoRows {
			panic(err)
		}

		ren.JSON(w, http.StatusOK, struct {
			Posts int64
			Views int64
		}{
			numPosts,
			numViews,
		})
	})
}

type post struct {
	// TODO: To Image url
	ImageID string
	Title   string
}

// TODO: Add paging
func postsHandler(db *sql.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		rows, err := db.Query("SELECT `id`, `title` FROM `posts` ORDER BY `createTime` DESC")
		if err != nil {
			panic(err)
		}

		posts := make([]post, 0)
		for rows.Next() {
			var id int64
			var title string

			if err := rows.Scan(&id, &title); err != nil {
				panic(err)
			}

			posts = append(posts, post{
				strconv.FormatInt(id, 10),
				title,
			})
		}

		ren.JSON(w, http.StatusOK, posts)
	})
}

// TODO: More compression...or?
// TODO: for SEO, url should be for meaningful, like /error.jpg, not /0
func imageHandler(routerName string, db *sql.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		id := mux.Vars(req)[routerName]
		if len(id) <= 0 {
			// TODO: NotFound handler
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// TODO: Caching
		var data []byte
		var fileName string
		if err := db.QueryRow("SELECT `image`, `fileName` FROM `posts` WHERE `id`=?", id).Scan(&data, &fileName); err != nil {
			if err == sql.ErrNoRows {
				// TODO: NotFound handler
				w.WriteHeader(http.StatusNotFound)
				return
			}
			panic(err)
		}

		ext := filepath.Ext(fileName)
		switch ext {
		case ".jpg", ".jpeg":
			w.Header().Set("CONTENT-TYPE", "image/jpeg")
			break
		case ".png":
			w.Header().Set("CONTENT-TYPE", "image/png")
			break
			//TODO: gif
		}

		w.Write(data)
	})
}

func imageDimansion(raws []byte, ext string) (w, h int, err error) {
	switch ext {
	case ".jpg", ".jpeg":
		config, err := jpeg.DecodeConfig(bytes.NewReader(raws))
		if err != nil {
			return 0, 0, err
		}
		return config.Width, config.Height, nil
	case ".png":
		config, err := png.DecodeConfig(bytes.NewReader(raws))
		if err != nil {
			return 0, 0, err
		}
		return config.Width, config.Height, nil
	}

	return 0, 0, errors.New("Unknown Ext")
}

// TODO: Limit paramters should be exported
func uploadHandler(db *sql.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.ContentLength >= (2<<25) || req.ContentLength == 0 {
			// TODO: Forbidden
			ren.JSON(w, http.StatusForbidden, responseError{"Invalid file size, up to 20mb"})
			return
		}

		if err := req.ParseMultipartForm(2 << 25); err != nil {
			panic(err)
		}

		title := req.FormValue("title")
		file, header, err := req.FormFile("image")
		if err != nil {
			panic(err)
		}
		defer file.Close()

		if header.Size >= 20000000 {
			// TODO: Forbidden and more message
			ren.JSON(w, http.StatusForbidden, responseError{"Invalid file size, up to 20mb"})
			return
		}

		fileName := header.Filename
		ext := strings.ToLower(filepath.Ext(fileName))
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
			ren.JSON(w, http.StatusForbidden, responseError{"Invalid file type. Only support JPG or PNG."})
			return
		}

		raws, err := ioutil.ReadAll(file)
		if err != nil {
			panic(err)
		}

		width, height, err := imageDimansion(raws, ext)
		if err != nil {
			ren.JSON(w, http.StatusForbidden, responseError{"Invalid file type. Only support JPG or PNG."})
			return
		} else if width > 1920 || height > 1080 {
			ren.JSON(w, http.StatusForbidden, responseError{"Invalid image size. upto 1920 x 1080."})
			return
		}

		res, err := db.Exec("INSERT INTO `posts`(`title`, `image`, `fileName`)VALUES(?, ?, ?)", title, raws, fileName)
		if err != nil {
			panic(err)
		}

		rid, err := res.LastInsertId()
		if err != nil {
			panic(err)
		}

		ren.JSON(w, http.StatusOK, rid)
	})
}

func exportHandler(name string, db *sql.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		buf := bytes.NewBuffer(nil)
		buf.WriteString("Title,FileName\n")

		rows, err := db.Query("SELECT `title`, `fileName` FROM `posts`")
		if err != nil {
			panic(err)
		}

		for rows.Next() {
			var title, fileName string
			if err := rows.Scan(&title, &fileName); err != nil {
				panic(err)
			}

			buf.WriteString(title)
			buf.WriteString(",")
			buf.WriteString(fileName)
			buf.WriteString("\n")
		}

		w.Header().Set("CONTENT-TYPE", "text/csv")
		w.Header().Set("CONTENT-DISPOSITION", `attachment; filename="`+name+`"`)
		w.Write(buf.Bytes())
	})
}

func viewStasHandler(db *sql.DB, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// TODO: Tx may slow...
		// TODO: Use channel here, if huge requests, may cause goroutines peak
		go func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Fprintf(os.Stderr, "View update failed, %v", r)
				}
			}()

			tx, err := db.Begin()
			if err != nil {
				panic(err)
			}
			defer tx.Rollback()

			if r, err := tx.Exec("UPDATE `stats` SET `view` = `view` + 1"); err != nil {
				panic(err)
			} else if i, err := r.RowsAffected(); err != nil {
				panic(err)
			} else if i <= 0 {
				if _, err := tx.Exec("INSERT INTO `stats`(`view`)VALUES(1)"); err != nil {
					panic(err)
				}
			}
			if err := tx.Commit(); err != nil {
				panic(err)
			}
		}()

		next.ServeHTTP(w, req)
	})
}

func fileHandler(filePath string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// TODO: Make file cache or file watch
		f, err := os.Open(filePath)
		if err != nil {
			panic(err)
		}
		defer f.Close()

		b, err := ioutil.ReadAll(f)
		if err != nil {
			panic(err)
		}

		// TODO: Parse file's extension
		w.Header().Set("CONTENT-TYPE", "text/html")
		w.Write(b)
	})
}
