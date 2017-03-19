package main

import (
	"encoding/json"
	"strconv"
	"log"
	"fmt"
	"flag"
	"net/http"
	"strings"

	"github.com/boltdb/bolt"
	b58 "github.com/itchyny/base58-go"
)

const bucketName string = "urlcutter"
const shiftAmount int = 3364 // 211
const postFormKey string = "url"

var db *bolt.DB

type Url struct {
	TargetUrl string
}

// Connect to the database
func DbConnect(d string) {
	var err error
	if db, err = bolt.Open(d, 0600, nil); err != nil {
		log.Fatalln("Bolt Driver Error: ", err)
	}
}

func DbClose() {
	db.Close()
}

// Update makes a modification to Bolt
func DbAddEntry(dataStruct interface{}) error {
	err := db.Update(func(tx *bolt.Tx) error {
		// Create the bucket
		bucket, e := tx.CreateBucketIfNotExists([]byte(bucketName))
		if e != nil {
			return e
		}

		// Encode the record
		encodedRecord, e := json.Marshal(dataStruct)
		if e != nil {
			return e
		}

		// Take Bolt's next sequience id (uint64)
		// and convert it to Base58 encoded string
		id, _ := bucket.NextSequence()
		// TODO use shiftAmount
		key := IntToBase58str(int(id))
		fmt.Println("The answer is:", key)

		// Store the record
		if e = bucket.Put([]byte(key), encodedRecord); e != nil {
			return e
		}
		return nil
	})
	return err
}

// View retrieves a record in Bolt
func DbGetEntry(key string, dataStruct interface{}) error {
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
		e := json.Unmarshal(v, &dataStruct)
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
	err := DbGetEntry(r.URL.Path[1:], &url)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
	}

	// Declare url and assign input from request
	url := &Url{
		TargetUrl: v,
	}

	// Add url to db
	err = DbAddEntry(&url)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

func main() {
	// Parse cli flags
	// urlcutter -dbpath foo.db -listen 0.0.0.0:8080
	dbpathPtr := flag.String("dbpath", "urlcutter.db", "File path for Bolt database. Created automatically if does not exist.")
	listenPtr := flag.String("listen", ":8080", "TCP socket to listen on. For example, 0.0.0.0:8080")
	flag.Parse()

	// Connect to Bolt
	DbConnect(*dbpathPtr)
	defer DbClose()

	// Play with Bolt
	//url := &Url{
	//	TargetUrl: "http://example.com",
	//}
	//DbAddEntry(&url)
	//url1 := Url{}
	//DbGetEntry("e", &url1)
	//fmt.Println("Entry:", url1.TargetUrl)

	http.HandleFunc("/", redirectHandler)
	http.HandleFunc("/create", createHandler)
	http.HandleFunc("/urlcutter", indexHandler)
	// Get config and serve
	http.ListenAndServe(*listenPtr, nil)
}
