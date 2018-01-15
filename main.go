package main

import (
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"net/http"
	"path/filepath"

	"github.com/boltdb/bolt"
	"github.com/speps/go-hashids"
)

var (
	host = flag.String("host", "", "Hostname")
	port = flag.String("port", "8080", "Port to listen on")

	salt      = flag.String("salt", "", "Salt for generating short URLs")
	dbPath    = flag.String("db", "magcargo.db", "Path to database file")
	minLength = flag.Int("minlength", 5, "Minimum length for short URLs")
)

func shortenURL(db *bolt.DB, hid *hashids.HashID, salt, URL string) (short string, err error) {
	err = db.Update(func(tx *bolt.Tx) (err error) {
		// open a bucket namespaced to salt
		b, err := tx.CreateBucketIfNotExists([]byte(salt))
		if err != nil {
			return
		}

		// get the next sequence number
		next, err := b.NextSequence()
		if err != nil {
			return
		}

		// convert it to a hashid
		s, err := hid.EncodeInt64([]int64{int64(next)})
		if err != nil {
			return
		}

		// save it to the db
		err = b.Put([]byte(s), []byte(URL))
		if err != nil {
			return
		}

		// update return value
		short = s
		return
	})

	return
}

func unshortenURL(db *bolt.DB, salt, short string) (URL string) {
	_ = db.View(func(tx *bolt.Tx) (err error) {
		b := tx.Bucket([]byte(salt))
		if b == nil {
			return
		}

		long := b.Get([]byte(short))
		URL = string(long)
		return
	})

	return
}

func createHandler(db *bolt.DB, hid *hashids.HashID, salt string) http.Handler {
	handler := func(res http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case http.MethodGet:
			short := req.URL.Path[1:]
			URL := unshortenURL(db, salt, short)
			if URL == "" {
				http.NotFound(res, req)
				return
			}

			http.Redirect(res, req, URL, http.StatusSeeOther)
		case http.MethodPost:
			err := req.ParseForm()
			if err != nil {
				log.Println(err)
				res.WriteHeader(http.StatusBadRequest)
				return
			}

			URL := req.Form.Get("url")
			if URL == "" {
				res.WriteHeader(http.StatusBadRequest)
				return
			}

			short, err := shortenURL(db, hid, salt, URL)
			if err != nil {
				log.Println(err)
				res.WriteHeader(http.StatusInternalServerError)
				return
			}

			res.WriteHeader(http.StatusCreated)
			res.Write([]byte(short))
		}
	}

	return http.HandlerFunc(handler)
}

func generateRandomSalt(length int) (salt string, err error) {
	b := make([]byte, length)
	_, err = rand.Read(b)
	if err != nil {
		return
	}

	salt = base64.StdEncoding.EncodeToString(b)
	return
}

func main() {
	flag.Parse()

	if *salt == "" {
		fmt.Println("Generating random salt...")

		// the number of bytes to generate the salt was arbitrarily decided
		randSalt, err := generateRandomSalt(12)
		if err != nil {
			log.Fatal(err)
		}

		*salt = randSalt
		fmt.Println("Using random salt: " + randSalt)
	}

	dbPathAbs, err := filepath.Abs(*dbPath)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Using database file at: " + dbPathAbs)

	db, err := bolt.Open(dbPathAbs, 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	hd := hashids.NewData()
	hd.Salt = *salt
	hd.MinLength = *minLength
	hid, err := hashids.NewWithData(hd)
	if err != nil {
		log.Fatal(err)
	}

	handler := createHandler(db, hid, *salt)
	addr := fmt.Sprintf("%s:%s", *host, *port)
	fmt.Println("Listening on " + addr)
	log.Fatal(http.ListenAndServe(addr, handler))
}
