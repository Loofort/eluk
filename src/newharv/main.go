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
			// suppose if ctx is done , getLinks returns immediatly with error
			links, err := ggl.getLinks(ctx, key, i)

			if err != nil {
				if err == context.Canceled {
					return
				}
				errc <- err
				break
			}

			select {
			case <-ctx.Done():
				return
			case out <- LinksBlob(links, key):
			}

		}
	}
}

func glinksWorker3(ctx context.Context, in <-chan string, out chan<- LinksBlob, errc chan<- error) {
	for key := range in {

		// query 10 pages one by one, without concurrency in order to protect from ban
		for i := 0; i < 10; i++ {
			links, err := ggl.getLinks(ctx, key, i)

			select {
			case <-ctx.Done():
				return
			default:
			}

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

func glinksWorker1(ctx context.Context, in <-chan string, out chan<- LinksBlob, errc chan<- error) {
	i, o, e := in, out, errc
	o, e = nil, nil
	var err error
	var lblob LinksBlob
	lo := make(chan LinksBlob, 1)
	lo_ := lo
	lo = nil
	le := make(chan error, 1)
	le_ := le
	le = nil

	for {
		select {
		case key, ok := <-i:
			if !ok {
				return
			}
			i = nil

			go func() {
				defer func() { i = in }()

				for p := 0; p < 10; p++ {
					links, err := ggl.getLinks(ctx, key, p)
					if err != nil {
						le <- err
						return
					}
					lo <- LinksBlob{links, key}
				}
			}()
		case err = <-le:
			le = nil
			e = errc
		case lblob = <-lo:
			lo = nil
			o = out
		case o <- lblob:
			lo = lo_
		case e <- err:
			le = le_
		case <-ctx.Done():
			return

		}
	}
}

func glinksWorker2(ctx context.Context, in <-chan string, out chan<- LinksBlob, errc chan<- error) {
	ec := make(chan error)
	go func() {
		for e := range ec {
			errc <- e
		}
	}()

	oc := make(chan LinksBlob)
	go func() {
		for o := range oc {
			out <- o
		}
	}()

	c := make(chan struct{})
	go func() {
		for key := range in {
			// query 10 pages one by one, without concurrency in order to protect from ban
			for i := 0; i < 10; i++ {
				links, err := ggl.getLinks(ctx, key, p)
				if err != nil {
					ec <- err
					break
				}
				oc <- LinksBlob{links, key}
			}
		}
		close(c)
	}()

	go func() {
		select {
		case <-c:
		case <-ctx.Done():
			oc1 := oc
			oc = make(chan LinksBlob, 1)
			close(oc1)

			///!!! error - we can't rewrite chan with out lock
		}
	}()
}

// pattern with two type of goroutines, 1 - payload; 2- merger
func best() {
	g1 := func(ctx context.Context, in <-chan string, errc chan<- error) chan<- LinksBlob {
		out := make(chan LinksBlob)
		go func() {
			for key := range in {
				for i := 0; i < 10; i++ {
					links, err := web.getLinks(ctx, key, i)
					if err != nil {
						errc <- err
						break
					}
					out <- LinksBlob{links, key}
				}
			}
			close(out)
		}()
		return out
	}

	g2 := func(ctx context.Context, n int, ins ...chan<- LinksBlob) chan<- LinksBlob {
		out = make(chan LinksBlob, n)
		output := func(in <-chan LinksBlob, out <-chan LinksBlob) {
			for o := range in {
				select {
				case <-ctx.Done():
				default:
					select {
					case <-ctx.Done():
					case out <- o:
					}
				}
			}

			for {
				select {
				case <-ctx.Done():
				case o, ok := <-in:
					if !ok {
						break
					}
					select {
					case <-ctx.Done():
					case out <- o:
					}
				}
			}

			in_ := in
			for {
				select {
				case <-ctx.Done():
					out = nil
				case o, ok := <-in:
					if !ok {
						break
					}
					in = nil
				case out <- o:
					in = in_
				}
			}

		}()
		return out
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
