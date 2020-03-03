package main

import (
	"context"
	"fmt"
	"os"
	"io/ioutil"
	"html/template"
	"net/http"
	"log"
	"path/filepath"
  "github.com/jackc/pgx"
)

type Video struct {
	Title string
	Link string
	Source string
	Owner User
}

type User struct {
	ID int
	Username string
	Link string
}

func (v Video) save() {
	conn, err := pgx.Connect(context.Background(), "postgres://admin:password@localhost/mydb")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to the database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())

	tag, err2 := conn.Exec(context.Background(), "insert into videos (title, link, src, owner_id) values ($1, $2, $3, $4)",
		v.Title, v.Link, v.Source, v.Owner.ID,
	)
	fmt.Println(tag)
	if err2 != nil {
		fmt.Fprintf(os.Stderr, "Query failed: %v\n", err2)
		os.Exit(1)
	}
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := pgx.Connect(context.Background(), "postgres://admin:password@localhost/mydb")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to the database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())

	rows, err2 := conn.Query(context.Background(), "select title, videos.link, src, users.id, username, users.link from videos inner join users on videos.owner_id = users.id")
	if err2 != nil {
		fmt.Fprintf(os.Stderr, "Query failed: %v\n", err2)
		os.Exit(1)
	}

	var videos []Video
	for rows.Next() {
		var videoTitle string
		var videoLink string
		var videoSrc string
		var userId int
		var userName string
		var userLink string

		err2 = rows.Scan(&videoTitle, &videoLink, &videoSrc, &userId, &userName, &userLink)
		if err2 != nil {
			log.Fatal(err2)
		}

		var videoOwner User
		videoOwner = User{userId, userName, userLink}
		videos = append(videos, Video{videoTitle, videoLink, videoSrc, videoOwner})
	}

	if rows.Err() != nil {
		log.Fatal(err2)
	}

	t, _ := template.ParseFiles("templates/index.htm")
	t.Execute(w, videos)
}

func uploadFile(w http.ResponseWriter, r *http.Request) string {
	r.ParseMultipartForm(1024 << 20)
	file, _, err := r.FormFile("file")
	if err != nil {
		fmt.Println(err)
	}
	defer file.Close()

	tempFile, err := ioutil.TempFile("tmp", "upload*.mp4")
	if err != nil {
		fmt.Println(err)
	}
	defer tempFile.Close()

	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Println(err)
	}
	tempFile.Write(fileBytes)
	return filepath.Base(tempFile.Name())
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
			filename := uploadFile(w, r)
			var newVid Video
			newVid.Source = filename
			r.ParseForm()
			newVid.Title = r.Form.Get("title")
			newVid.Link = r.Form.Get("description")
			newVid.Owner = User{ID: 1,}
			newVid.save()
		} else {
			t, _ := template.ParseFiles("templates/upload.htm")
			t.Execute(w, Video{})
		}
	}

	func main() {
		http.Handle("/tmp/", http.StripPrefix("/tmp/", http.FileServer(http.Dir("./tmp"))))
		http.HandleFunc("/upload", uploadHandler)
		http.HandleFunc("/", homeHandler)
		log.Fatal(http.ListenAndServe(":8080", nil))
	}
