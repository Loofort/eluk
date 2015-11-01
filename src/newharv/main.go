// Architecture note:
// Pipeline arhitecture, the flow is going from stage to stage
// Each stage can rise an error.
// Each stage has separate error handling that able to cancel current stage and all upstream

package main

import (
	"./db"
	"./reader"
	"./web"
	"flag"
	"github.com/golang/glog"
	"golang.org/x/net/context"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// global context parrent for each context
var gx, gcancel = context.WithCancel(context.Background())

func main() {
	flag.Parse()
	c := func() {}

	// read keyword from stdin
	ctx, cancel := context.WithCancel(gx)
	keyc, errc := stdinStage(ctx)
	c = errHanler(c, cancel, errc, "read kewrods")

	// obtains links from google, just one thread
	ctx, cancel = context.WithCancel(gx)
	serpc, errc := serpStage(ctx, keyc)
	c = errHanler(c, cancel, errc, "get links")

	// create shop
	shopc, c := create(serpc, c)

	// check if grabbed links is really a shop
	checkedc, c := stage(checkStage, shopc, c, "check shop")

	// gather emails from site
	gatheredc, _ := stage(emailStage, checkedc, c, "gather email")
	//stage(emailStage, checkedc, c, "gather email")

	for range gatheredc {
	}
}

// create shops object, save it to db
func create(serpc <-chan Serp, cancel func()) (<-chan Shop, func()) {
	// creates shops
	// this goroutine is not cancelable, no ctx is needed
	shopc := make(chan Shop, 100)
	go func() {
		defer close(shopc)
		for serp := range serpc {
			for _, link := range serp.links {
				shop, err := NewShop(link, serp.key)
				if err != nil {
					glog.Errorf("can't parse shop link %s: %v", link, err)
					continue
				}
				shopc <- shop
			}
		}
	}()

	// save shops in db, checks for unique
	savedc, cancel := stageDB(shopc, cancel, "save shop")
	return savedc, cancel
}

// helper function. Starts stage that has in and out channel type of Shop
func stage(worker func(context.Context, <-chan Shop) (<-chan Shop, <-chan error), in <-chan Shop, cancel func(), name string) (<-chan Shop, func()) {
	ctx, cncl := context.WithCancel(gx)
	out, errc := worker(ctx, in)
	cancel = errHanler(cancel, cncl, errc, name)

	// save to db
	//out, cancel := stageDB(out, cancel, name+", save")
	return out, cancel
}

// inherits context from DB gloabl contex.
// Executes Upsert worker
func stageDB(in <-chan Shop, cancel func(), name string) (<-chan Shop, func()) {
	ctx, cncl := context.WithCancel(gx)
	out, errc := upsertStage(ctx, in)
	// if database throws an error whole program should be terminated
	errHanler(gcancel, cncl, errc, name)
	// but global cancel shouldn't be executed in upstream cancelation event.
	cancel = addCancel(cancel, cncl)
	return out, cancel
}

// runs new error handler for a Stage in goroutine,
// returuns cancel func that able to cancel all previous stage
// In error case handler executes cancel func
func errHanler(cancel, cancelStage func(), errc <-chan error, name string) func() {
	cancel = addCancel(cancel, cancelStage)

	go func() {
		for err := range errc {
			if err == context.Canceled {
				glog.Infof("[%s] worker is canceled", name)
				continue
			}

			glog.Errorf("[%s] is going to stop because of err: %v", name, err)
			cancel()
		}
		glog.Infof("[%s] stoped", name)
	}()

	return cancel
}

// sums two cancle funcs
func addCancel(cancel, cancelStage func()) func() {
	return func() {
		cancel()
		cancelStage()
	}
}

// check if error is thrown by context
func ctxErr(err error) bool {
	return (err == context.Canceled || err == context.DeadlineExceeded)
}

// ################################### STAGES HANDLERS ############################################

// upsert shop in db, if it doesn't exist - insert.
// unlimited count of worker (limited by mgo pool)
func upsertStage(ctx context.Context, in <-chan Shop) (<-chan Shop, <-chan error) {
	out := make(chan Shop)
	errc := make(chan error)
	go func() {
		defer close(out)
		defer close(errc)

		var wg sync.WaitGroup
		for shop := range in {
			//spawn new goroutine to create a shop
			wg.Add(1)
			go func() {
				defer wg.Done()

				err := db.Upsert(ctx, shop)
				if db.IsDub(err) {
					return
				}

				if err != nil {
					errc <- err
					return
				}

				select {
				case out <- shop:
				case <-ctx.Done():
					errc <- ctx.Err()
				}
			}()
		}

		wg.Wait()
	}()

	return out, errc
}

// stdinStage
func stdinStage(ctx context.Context) (<-chan string, <-chan error) {
	out := make(chan string)
	errc := make(chan error)

	go func() {
		defer close(out)
		defer close(errc)
		lr := reader.NewLineReader(os.Stdin)
		for {
			key, err := lr.Next(ctx)
			if err == io.EOF {
				glog.Info("No more keywords")
				return
			}

			if err != nil {
				errc <- err
				return
			}

			select {
			case out <- key:
				glog.V(1).Infof("input key: %s\n", key)
			case <-ctx.Done():
				errc <- ctx.Err()
				return
			}
		}
	}()
	return out, errc
}

// serp - Search Engine Result Page
type Serp struct {
	links []string
	key   string
}

// one thread, bacause google doesn't like load activity
func serpStage(ctx context.Context, in <-chan string) (<-chan Serp, <-chan error) {
	out := make(chan Serp)
	errc := make(chan error)

	go func() {
		defer close(out)
		defer close(errc)

		// don't check cancelation on IN
		for key := range in {
			var last string
			var i int
			for i = 0; i < 10; i++ {
				links, err := web.GLinks(ctx, key, i)
				if err != nil {
					errc <- err
					break
				}

				if len(links) == 0 {
					glog.Infof("no search results for keyword: %s", key)
					break
				}

				// if we got the same urls it means we on last result page
				// and have to skip any pending queries
				if i > 0 && last == links[len(links)-1] {
					break
				}
				last = links[len(links)-1]

				// need to make pause for google
				pause := time.After(2 * time.Second)

				select {
				case out <- Serp{links, key}:
					glog.V(1).Infof("for key=\"%s\" %d page got links: \n\t%s\n", key, i, strings.Join(links, "\n\t"))
				case <-ctx.Done():
					errc <- ctx.Err()
					return
				}

				select {
				case <-pause:
				case <-ctx.Done():
					errc <- ctx.Err()
					return
				}

			}

			glog.Infof("for keyword %s SE found %d pages", key, i)
		}
	}()

	return out, errc
}

// try to find a signal word on shop page
// pool of threads
func checkStage(ctx context.Context, in <-chan Shop) (<-chan Shop, <-chan error) {
	out := make(chan Shop)
	errc := make(chan error)

	var wg sync.WaitGroup
	workerCount := 100
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					errc <- ctx.Err()
					return

				// read IN channel
				case shop, ok := <-in:
					if !ok {
						return
					}

					ok, err := web.IsShop(ctx, shop.Link)
					// if error with site (or net) skip it, but rise a signal if it's context's error
					if ctxErr(err) {
						errc <- err
						return
					}
					if err != nil {
						glog.Errorf("can't check shop %s: %v", shop.Link, err)
						continue
					}

					if !ok {
						glog.Infof("Site %s is not a shop", shop.Host)
						shop.Invalid = true
					}

					shop.Stage = STG_CHECKED
					select {
					case out <- shop:
					case <-ctx.Done():
						errc <- ctx.Err()
					}
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(out)
		close(errc)
	}()

	return out, errc
}

// on this stage we crawl site for one layer of depth.
// we limit crawlers per one site up to 5 to do not overload crawled site.
// from our side we have two type of resource - bandwidth and cpu. our performance will be limited by one or both of this resources and it's likely be bandwidth.
// we should limit overall amount of goroutines to be protected from net channel overload and increasing latency.
func emailStage(ctx context.Context, in <-chan Shop) (<-chan Shop, <-chan error) {
	out := make(chan Shop)
	errc := make(chan error)

	go func() {
		defer close(out)
		defer close(errc)

		// for each shop start crawler
		for shop := range in {
			if shop.Invalid {
				continue
			}
			// to avoid blocking on queuing new link and to avoid bulk of goroutines
			// we need intermidiate goroutine to handle links queue.
			// crawler has two queues - to pop and push urls
			popc := make(chan string, 100)
			pushc := make(chan []string, 100)
			go func() {

				link := shop.Link
				tasks := make([]string, 0, 100)
				tasksc := popc

				for {
					select {
					case links, ok := <-pushc:
						if !ok {
							return
						}
						if len(links) == 0 {
							continue
						}

						tasks := append(tasks, links...)
						if tasksc == nil {
							tasksc = popc
							link, tasks = tasks[0], tasks[1:]
						}

					case tasksc <- link:
						if len(tasks) == 0 {
							tasksc = nil
							continue
						}
						link, tasks = tasks[0], tasks[1:]
					}
				}

			}()

			// start several workers per each crawler
			for i := 0; i < 2; i++ {
				go func() {

					// wroker obtains url from queue, requests and parses page, and pushes new links to queue
					for link := range popc {
						links, err := web.Links(ctx, link)

						// ignore error that are related to http workflow
						if ctxErr(err) {
							errc <- err
							return
						}
						if err != nil {
							glog.Errorf("crawler can't get page %s: %v", link, err)
						}

						pushc <- links
					}
				}()
			}
		}
	}()

	return out, errc
}
