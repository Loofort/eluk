// Architecture note:
// scope of processes of the same type is pool
// live circle of each task is a flow transfering from pool to pool
//
// there is a god that can interact with each pool directly.
// the god can receive error/done signal from pool and ask other pool to finish 
//
package main

func main() {
// pool to 

// pool to obtain links from google
shopsc := make(chan Shops, )
ctx, gcancel = context.WithCancel(context.Background())
go glinksBlob(ctx, keysc, shopsc, gerrc)


// the god runtime
for {
	select {
	case err := <-gerrc
		gcancel()

	}

}



}

func glinksBlob (ctx context.Context, in <-chan string, out chan<- shop, errc chan<- error) {
	for key := range in {

		// query 10 pages one by one, without concurrency in order to protect from ban
		for i:=0; i<10; i++ {
			links, err := getLinks(ctx, key, i)

					// make local copy of chans
			out := out
			errc := errc 

			//turn of one of channel
				if err == nil {
					errc = nil
				} else {
					out = nil
				}

				select {
				case <-ctx.Done():
					return
				case errc <- err:
				case out <- links:
				}

		}
	}
}




func glinksBlob1 (ctx context.Context, in <-chan string, out chan<- shop, errc chan<- error) {
	keyc := in
	linkc := out
	errorc := errc
	linkc, errorc = nil,nil

	for {

		select {
		case key, ok := <-in:
			if !ok {
				return
			}
			in = nil

			gtaskc = make(chan gtask, 10)
			for i:=0; i<10 ; i++ {
				gtaskc <- gtask{key, i}
			}
			close(gtaskc)

		case gt, ok := <- gtaskc:
			if !ok {
				in = keyc
				break
			}

			links, err = getLinks(gt.key, gt.i)

			if err != nil {
				eout = errc 
			} else {
				out = nil
			}


				
				
			

		case <-ctx.Done()
			return
		}
	}


}




func main() {

	for {
		select {
		case kwd := <-keywordCh:
			// turn off this case till current will be done
			keywordCh = nil

			go func() {
				// create 10 url to query
				gurlCh := make(chan string, 10)
				for p := 0; p < 10; p++ {
					gurlCh <- gurl(kwd, p)
				}

				for gurl := range gurlCh {
					links := gLinks(gurl)

					for _, l := range links {
						linkCh <- l
					}

					// turn on link parser for next keyword
					keywordCh = keywordChan
				}
			}()
		case link := <-linkCh:

		}
	}
}
