package gueuer

import (
	"io"
	"log"
	"net/http"
	"time"
)

const (
	SERVER    = "localhost:12345"
	PATH      = "/queuer_test_page"
	PAGE_SIZE = 1024
	PAGE_BYTE = 100
)

var (
	page [PAGE_SIZE]byte
	load Duration
)

func init() {
	for i := 0; i < PAGE_SIZE; i++ {
		page[i] = PAGE_BYTE
	}

	http.HandleFunc(PATH, func(w http.ResponseWriter, req *http.Request) { w.Write(page) })
	go log.Fatal(http.ListenAndServe(SERVER, nil))
	go initQueuer()
}

func initQueuer() {
	url = "http://" + SERVER + PATH
	firstGet(10, url)

	//get initial average speed for keep-alive connection
	var elapsed Duration
	for i := 0; i < 10; i++ {
		start := time.Now()
		resp, err := http.Get(url)
		body, err := ioutil.ReadAll(resp.Body)
		elapsed += time.Since(start)
		resp.Body.Close()
	}

	elapsed
}

func firstGet(tries int, url string) {
	var err error
	for i := 0; i < tries; i++ {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			return
		}
	}
	log.Fatalf("can't obtain Queuer Test Page: ", err)
}

func run() {

}
