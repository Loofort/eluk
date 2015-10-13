package main

// abstractStage implements commmon cancelation pattern.
// It creates Workers (for this case just one) - workker is the wrapper for payload to run it asyncroniusly.
// And all workers channels are merged in one by Merger.
// If you cancel the Stage the Merger just turns off transfer from Workers and waits till inbound channel will be closed.
func abstractStage(ctx context.Context, cancel func()) chan<- LinksBlob {
	linksc1, errc := abstractWorker(ctx, keyc)
	linksc := abstractMerger(ctx, linksc1)
	go func() {
		// errors might occures multiple time before worker will be actually canceled
		// cancel() is safe to be called multiple time
		for err := range errc {
			cancel()
		}
	}()
	return linksc
}

// Run Payload async. creat out chanel, close it when goroutine gets done.
func abstractWorker(ctx context.Context, in <-chan string) (chan<- LinksBlob, chan<- error) {
	out := make(chan LinksBlob, 10)
	errc := make(chan error)

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
		close(errc)
	}()
	return out, errc
}

func abstractMerger(ctx context.Context, n int, ins ...chan<- LinksBlob) chan<- LinksBlob {
	out = make(chan LinksBlob, n)
	var wg sync.WaitGroup
	output := func(in <-chan LinksBlob) {
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
		wg.Done()
	}()

	for _, in := range ins {
		output(in)
	}
	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}
