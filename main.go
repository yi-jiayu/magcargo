package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/boltdb/bolt"
	"github.com/speps/go-hashids"
)

var (
	db   *bolt.DB
	salt string
	hid  *hashids.HashID
)

// itob returns an 8-byte big endian representation of v.
func itob(v int) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func newShortURL(url string) (id int, err error) {
	err = db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("urls"))
		if err != nil {
			return err
		}

		next, _ := b.NextSequence()
		id = int(next)

		return b.Put(itob(id), []byte(url))
	})

	return id, err
}

func idToString(id int) string {
	s, _ := hid.Encode([]int{id})
	return s
}

func getShortURL(id int) (url string) {
	_ = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("urls"))
		if b == nil {
			return nil
		}

		burl := b.Get(itob(id))
		url = string(burl)
		return nil
	})

	return url
}

func stringToID(short string) (id int, err error) {
	d, err := hid.DecodeWithError(short)
	if err != nil {
		return 0, err
	}

	return d[0], nil
}

func ServeHTTP(res http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		short := req.URL.Path[1:]
		id, err := stringToID(short)
		if err != nil {
			log.Println(err)
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		url := getShortURL(id)
		if url == "" {
			http.NotFound(res, req)
			return
		}

		http.Redirect(res, req, url, http.StatusSeeOther)
	case http.MethodPost:
		_ = req.ParseForm()
		url := req.Form.Get("url")
		id, err := newShortURL(url)
		if err != nil {
			log.Println(err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		short := idToString(id)
		res.WriteHeader(http.StatusCreated)
		res.Write([]byte(short))
	}
}

func main() {
	exe, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}

	cwd := filepath.Dir(exe)
	fmt.Println("Current directory: " + cwd)

	db, err = bolt.Open("magcargo.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	salt = "slugma magcargo swinub"
	hd := hashids.NewData()
	hd.Salt = salt
	hd.MinLength = 5
	hid, _ = hashids.NewWithData(hd)

	log.Fatal(http.ListenAndServe(":8080", http.HandlerFunc(ServeHTTP)))
}
