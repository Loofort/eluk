// Architecture note:
// scope of processes of the same type is pool
// live circle of each task is a flow transfering from pool to pool
//
// there is a manger that can interact with each pool directly.
// the manger can receive error/done signal from pool and ask other pool to finish
//
package main

func main() {

	keyc := make(chan string, 1)
	kerrc := make(chan error)
	ctx, kcancel = context.WithCancel(context.Background())
	go inputWorker(ctx, keyc, kerrc)

	// pool obtains links from google, just one thread
	linksc := make(chan LinksBlob, 1)
	gerrc := make(chan error)
	ctx, gcancel = context.WithCancel(context.Background())
	go glinksWorker(ctx, keyc, linksc, gerrc)

	// pool creates shops in db, checks for unique
	// unlimiited threads (infact limited by mgo pool size)
	shopc := make(chan Shop, 1)
	merrc := make(chan error)
	go shopCreatorsPool(ctx, linksc, shopc, merrc)

	// the god runtime
	// when we got fatal error from particular pool we need to
	//   cancel all pools befor,
	//   cancel all workers of current pool,
	//   and close current output channel, if no other pool is writing to it
	for {
		select {
		case err := <-gerrc:
			gcancel()

		}

	}

}

func glinksWorker(ctx context.Context, in <-chan string, out chan<- LinksBlob, errc chan<- error) {
	for key := range in {

		// query 10 pages one by one, without concurrency in order to protect from ban
		for i := 0; i < 10; i++ {

			e = make(chan err, 1)
			o = make(chan LinksBlob, 1)

			go func() {
				if links, err := ggl.getLinks(ctx, key, i); err != nil {
					e <- err
				} else {
					o <- links
				}
			}()

			select {
			case <-ctx.Done():
				return
			case errc <- <-e:
			case out <- <-o:
			}

			///////////////////////////

			d = make(chan struct{})
			go func() {
				if links, err := ggl.getLinks(ctx, key, i); err != nil {
					errc <- err
				} else {
					out <- links
				}
				close(d)
			}()

			select {
			case <-ctx.Done():
				errc = make(chan err, 1)
				out = make(chan LinksBlob, 1)
				return
			case <-d:
			}

			///////////////////////////
			links, err := ggl.getLinks(ctx, key, i)

			if err != nil {
				select {
				case errc <- err:
					continue
				case <-ctx.Done():
					return
				}
			}

			select {
			case out <- LinksBlob(links, key):
			case <-ctx.Done():
				return
			}

			///////////////////////

			links, err := ggl.getLinks(ctx, key, i)

			if err != nil {
				if err == context.Canceled {
					return
				}

				select {
				case <-ctx.Done():
					return
				case errc <- err:
					continue
				}
			}

			select {
			case <-ctx.Done():
				return
			case out <- LinksBlob(links, key):
			}

			///////////////////////

			links, err := ggl.getLinks(ctx, key, i)

			if err != nil {
				if err == context.Canceled {
					return
				}
				errc <- err
				continue
			}

			select {
			case <-ctx.Done():
				return
			case out <- LinksBlob(links, key):
			}

		}
	}
}

func glinksWorker(ctx context.Context, in <-chan string, out chan<- LinksBlob, errc chan<- error) {
	for key := range in {

		// query 10 pages one by one, without concurrency in order to protect from ban
		for i := 0; i < 10; i++ {
			links, err := ggl.getLinks(ctx, key, i)

			if err != nil {
				select {
				case errc <- err:
					continue
				case <-ctx.Done():
					return
				}
			}

			select {
			case out <- LinksBlob(links, key):
			case <-ctx.Done():
				return
			}
		}
	}
}

func shopCreatorsPool(ctx context.Context, in <-chan LinksBlob, out chan<- shop, errc chan<- error) {
	for blob := range in {
		for _, link = range blob.links {
			go func() {

				shop := NewShop(link, blob.key)
				ok, err := db.Save(ctx, shop)

				if err != nil {
					select {
					case errc <- err:
					case <-ctx.Done():
					}
					return
				}

				if ok {
					select {
					case out <- shop:
					case <-ctx.Done():
					}
				}
			}()
		}
	}
}
