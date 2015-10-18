// Architecture note:
// Pipeline arhitecture, the flow is going from stage to stage
// Each stage can rise an error.
// Each stage has separate error handling that able to cancel current stage and all upstream

package main

import (
	"golang.org/x/net/context"
)

// global context parrent for each context
var gx, gcancel = context.WithCancel(context.Background())

func main() {
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
	gatheredc, c := stage(emailStage, checkedc, c, "gather email")
}

// create shops object, save it to db
func create(serpc <-chan Serp, cancel func()) (<-chan Shop, func()) {
	// creates shops
	// this goroutine is not cancelable, no ctx is needed
	shopc := make(chan Shop, 100)
	go func() {
		defer close(shopc)
		for blob := range serpc {
			for _, link := range serp.links {
				shopc <- NewShop(link, serp.key)
			}
		}
	}()

	// save shops in db, checks for unique
	savedc, cancel := stageDB(shopc, cancel, "save shop")
	return savedc, cancel
}

// helper function. Starts stage that has in and out channel type of Shop
func stage(worker func() (chan Shop, chan error), in <-chan Shop, cancel func(), name string) (<-chan Shop, func()) {
	ctx, cncl := context.WithCancel(gx)
	out, errc := worker(ctx, in)
	cancel = errHanler(cancel, cncl, errc, name)

	// save to db
	dbout, cancel := stageDB(out, cancel, name+", save")
	return dbout, cancel
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

// ################################### STAGES HANDLERS ############################################

// upsert shop in db, if it doesn't exist - insert.
// unlimited count of worker (limited by mgo pool)
func upsertStage(ctx context.context, in <-chan Shop) (chan<- Shop, chan<- error) {
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

				ok, err := db.Upsert(ctx, shop)
				if db.IsDub(err) {
					return
				}

				if err != nil {
					errc <- err
					return
				}

				// do not foraward invalid shops
				if shop.Invalid {
					return
				}

				select {
				case out <- shop:
				case <-ctx.Done():
				}
			}()
		}

		wg.Wait()
	}()

	return out, errc
}

// stdinStage
func stdinStage(ctx context.Context) (chan<- string, chan<- error) {
	out := make(chan LinksBlob)
	errc := make(chan error)

	go func() {
		defer close(out)
		defer close(errc)
		lr := NewLineReader(os.Stdin)
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
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, errc
}

// serp - Search Engine Result Page
// one thread, bacause google doesn't like load activity
func serpStage(ctx context.Context, in <-chan string) (chan<- Serp, chan<- error) {
	out := make(chan<- Serp)
	errc := make(chan<- error)

	go func() {
		defer close(out)
		defer close(errc)

		// don't check cancelation on IN
		for key := range in {
			var last string
			var i int
			for i = 0; i < 10; i++ {
				links, err := web.getLinks(ctx, key, i)
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

				select {
				case out <- Serp{links, key}:
				case <-ctx.Done():
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
func sheckStage(ctx context.Context, in <-chan Shop) (chan<- Shop, chan<- error) {
	out := make(chan Shop)
	errc := make(chan error)

	var wg sync.WaitGroup
	workerCount := 100
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				// read IN channel
				select {
				case <-ctx.Done():
					return
				case shop, ok := <-in:
					if !ok {
						return
					}

					ok, err := web.isShop(ctx, shop.Link)
					if err != nil {
						errc <- err
						return
					}

					if !ok {
						glog.Infof("Site %s is not a shop", shop.Host)
						shop.Invalid = true
					}

					shop.Stage = STG_CHECKED
					select {
					case out <- shop:
					case <-ctx.Done():
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
func emailStage(ctx context.context, in <-chan Shop) (chan<- Shop, chan<- error) {

}
