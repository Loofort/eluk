package main

import (
	"bufio"
	"fmt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sync"
)

const MGO_COLLECTION = "shops"

var web = &http.Client{}
var mgodb = newMgo("juser:jpass@ds031613.mongolab.com:31613/junodb")
var linksRE = regexp.MustCompile(`<a href="/url\?q=([^&]+)`)
var domainRE = regexp.MustCompile(`https?://([^/]+)`)

type task interface{}

func init() {
	log.SetFlags(log.Lshortfile)
}

func db() (*mgo.Collection, func()) {
	sess := mgodb.Copy()
	release := func() {
		sess.Close()
	}
	return sess.DB("").C(MGO_COLLECTION), release
}
func newMgo(url string) *mgo.Session {
	// connect to mongo
	sess, err := mgo.Dial(url)
	if err != nil {
		panic(err)
	}

	// Optional. Switch the session to a monotonic behavior.
	sess.SetMode(mgo.Monotonic, true)
	c := sess.DB("").C("shops")

	// create indexes if don't exist
	indexes := []mgo.Index{
		// to control shop duplicates,
		mgo.Index{
			Key:        []string{"domain"},
			Unique:     true,
			DropDups:   false,
			Background: true,
			Sparse:     true,
		},
		// for fast stage access
		mgo.Index{
			Key:        []string{"stage"},
			DropDups:   false,
			Background: true,
			Sparse:     true,
		},
		/*
			// to get confirmed user
			mgo.Index{
				Key:        []string{"_id", "confirm"},
				Unique:     true,
				Background: true,
				Sparse:     true,
			},
		*/
	}

	for _, index := range indexes {
		if err := c.EnsureIndex(index); err != nil {
			panic(err)
		}
	}

	return sess
}

func main() {

	kwdCh := produceKeywords()
	gurlCh := multiplex(kwdCh, func(t task, out chan task) task {
		for i := 0; i < 10; i++ {
			out <- fmt.Sprintf("http://www.google.com/search?q=%s&gbv=2&gws_rd=ssl&num=100&start=%d00", t.(string), i)
		}
	})

	linkCh := tube(gurlCh, 3, func(t task) task {
		return
	})

	linkCh := produceLinks(kwdCh)

	savedCh := produceSavedlinks(linkCh)
	printChan(linkCh)
	/*
		savedCh := produceSavedlinks(linkCh)
		shopCh := produceFilteredShop(savedCh)
		mailCh := produceMails(shopCh)
	*/
}

func printChan(ch chan string) {
	for s := range ch {
		log.Printf("chan str: %v\n", s)
	}
}

func produceKeywords() chan string {
	kwdCh := make(chan interface{}, 1)
	go func() {
		buf := bufio.NewReader(os.Stdin)

		skip := false
		for {
			line, isPrefix, err := buf.ReadLine()
			if err != nil {
				log.Println("end reading stdin becouse of: ", err)
				close(kwdCh)
				break
			}
			//skip prefix
			if isPrefix {
				log.Println("skip to long line")
				skip = true
				continue
			}
			//skip postfix
			if skip {
				skip = false
				continue

			}

			log.Printf("got input keydord: %s", line)
			kwdCh <- string(line)
		}
	}()
	return kwdCh
}

func produceLinks(kwdCh chan string) chan string {
	linkCh := make(chan string, 1)
	go func() {
		wg := &sync.WaitGroup{}
		sem := make(chan int, 3)

		for kwd := range kwdCh {
			// extract links from 10 G pages
			for i := 0; i < 10; i++ {
				sem <- 1
				wg.Add(1)
				go func(page int) {
					defer wg.Done()
					log.Printf("start produce links for keword %s from page %d\n", kwd, page)

					links := linksFromG(kwd, page)
					for _, link := range links {
						linkCh <- link
					}
					<-sem
				}(i)
			}
		}
		log.Printf("no more keywords for links production\n")
		wg.Wait()
		close(linkCh)
	}()
	return linkCh
}

func linksFromG(kwd string, page int) []string {
	body := requestG(kwd, page)
	match := linksRE.FindAllStringSubmatch(body, -1)
	links := make([]string, 0, len(match))
	for _, m := range match {
		link, err := url.QueryUnescape(m[1])
		if err != nil {
			log.Printf("can't unescape url %s", m[1])
			continue
		}
		links = append(links, link)
	}
	return links
}

func requestG(kwd string, page int) string {
	url := fmt.Sprintf("http://www.google.com/search?q=%s&gbv=2&gws_rd=ssl&num=100&start=%d00", kwd, page)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("error when creating request for '%s': %v\n", kwd, err)
		return ""
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/44.0.2403.155 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Encoding", "gzip, deflate, sdch")
	req.Header.Set("Accept-Language", "ru-RU,ru;q=0.8,en-US;q=0.6,en;q=0.4,vi;q=0.2")
	req.Header.Set("Cache-Control", "max-age=0")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	resp, err := web.Do(req)
	if err != nil {
		log.Printf("error when request for '%s' page %d: %v\n", kwd, page, err)
		return ""
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("error when read response for '%s' page %d: %v\n", kwd, page, err)
		return ""
	}

	return string(body)
}

type worker func(task) task

func tube(in chan task, cnt int, work worker) chan task {
	out := make(chan task, 1)
	go func() {
		var sem = make(chan int, cnt)
		for task := range in {
			sem <- 1
			go func() {
				result := work(task)
				if result != "" {
					out <- result
				}
				<-sem
			}()
		}

		// loop ends in if incoming channel was closed
		// we have to wait till started workers are finish and then close outgoing channel
		for i := 0; i < cnt; i++ {
			sem <- 1
		}
		close(out)
	}()
	return out
}

func multiplex(in chan task, work worker) chan task {
	out := make(chan task, 1)
	go func() {
		for task := range in {
			result := work(task)
			for t := range result {
				out <- t
			}
		}
		close(out)
	}()
	return out
}

// -------------- task -------------------

type Stage int

const (
	STG_NEW Stage = iota
	STG_SHOP
	STG_MAIL
	STG_STAT
)

type Shop struct {
	ID     bson.ObjectId `bson:"_id"`
	Domain string
	Link   string
	Stage  Stage
}

func (s *Shop) String() string {
	return "[" + s.Domain + "]" + s.Link
}

func makeShop(link string) *Shop {
	col, release := db()
	defer release()

	match := domainRE.FindStringSubmatch(link)
	shop := &Shop{
		ID:     bson.NewObjectId(),
		Domain: match[1],
		Link:   link,
		Stage:  STG_NEW,
	}

	err := col.Insert(shop)
	if err == nil {
		return shop
	}

	if mgo.IsDup(err) {
		return nil
	}

	log.Panicf("can't save shop to db, err: %#v", err)
}
