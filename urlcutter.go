package main

import (
	"encoding/json"
	"fmt"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/boltdb/bolt"
	b58 "github.com/itchyny/base58-go"
)

const bucketName string = "urlcutter"
const shiftAmount int = 3364 // 211
const postFormKey string = "url"

var db *bolt.DB

type Url struct {
	Key string
	TargetUrl string
}

// Connect to the database
func DbConnect(d string) {
	var err error
	if db, err = bolt.Open(d, 0600, nil); err != nil {
		log.Fatalln("Bolt Driver Error", err)
	}
}

func DbClose() {
	db.Close()
}

// Add Url to Bolt
func AddUrl(u *Url) error {
	err := db.Update(func(tx *bolt.Tx) error {
		// Create the bucket
		bucket, e := tx.CreateBucketIfNotExists([]byte(bucketName))
		if e != nil {
			return e
		}

		// Take Bolt's next sequience id (uint64)
		// and convert it to Base58 encoded string
		id, _ := bucket.NextSequence()
		// TODO use shiftAmount
		key := IntToBase58str(int(id))

		// Update Url with resulted key
		u.Key = key

		// Encode the record
		encodedRecord, e := json.Marshal(u)
		if e != nil {
			return e
		}

		// Store the record
		if e = bucket.Put([]byte(key), encodedRecord); e != nil {
			return e
		}
		return nil
	})
	return err
}

// View retrieves a record in Bolt
func GetUrl(key string, u *Url) error {
	err := db.View(func(tx *bolt.Tx) error {
		// Get the bucket
		b := tx.Bucket([]byte(bucketName))
		if b == nil {
			return bolt.ErrBucketNotFound
		}

		// Retrieve the record
		v := b.Get([]byte(key))
		if len(v) < 1 {
			return bolt.ErrInvalid
		}

		// Decode the record
		e := json.Unmarshal(v, &u)
		if e != nil {
			return e
		}

		return nil
	})

	return err
}

func IntToBase58str(i int) string {
	// Int -> string -> Byte -> Base58 Byte -> string
	// TODO: Think on more optimal way
	bs := []byte(strconv.Itoa(i))
	encoding := b58.FlickrEncoding
	encoded, _ := encoding.Encode(bs)
	return string(encoded)
}

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	// Get Url object from database
	url := Url{}
	err := GetUrl(r.URL.Path[1:], &url)
	// TODO handle if key not found
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Redirect only if target url starts with http
	if strings.HasPrefix(url.TargetUrl, "http") {
		http.Redirect(w, r, url.TargetUrl, http.StatusMovedPermanently)
	}
}

func createHandler(w http.ResponseWriter, r *http.Request) {
	var err error

	// TODO: Handle redirects if not POST
	// Allow only POST method
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method allowed", http.StatusBadRequest)
		return
	}

	// Parse the request body
	err = r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get value from request and check if valid http url
	v := r.PostFormValue(postFormKey)
	if !strings.HasPrefix(v, "http") {
		http.Error(w, "Incorrect input", http.StatusBadRequest)
		return
	}

	// Declare url and assign input from request
	url := Url{
		TargetUrl: v,
	}

	// Add url to db
	err = AddUrl(&url)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Print resulted url
	fmt.Fprintf(w, url.Key)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	// TODO Use real template
	b, err := ioutil.ReadFile("template/index.html")
	if err != nil {
		log.Fatalln("Error reading template", err)
	}
	fmt.Fprintf(w, string(b))
}

func main() {
	// Parse cli flags
	// Example: urlcutter -dbpath foo.db -listen 0.0.0.0:8080
	dbpathPtr := flag.String("dbpath",
		"urlcutter.db",
		"File path for Bolt database. Created automatically if does not exist.")
	listenPtr := flag.String("listen",
		":8080",
		"TCP socket to listen on. For example, 0.0.0.0:8080")
	flag.Parse()

	// Connect to Bolt
	DbConnect(*dbpathPtr)
	defer DbClose()

	// Handlers
	http.HandleFunc("/", redirectHandler)
	http.HandleFunc("/create", createHandler)

	// Entry handler
	http.HandleFunc("/urlcutter", indexHandler)

	// Get config and serve
	http.ListenAndServe(*listenPtr, nil)
}
