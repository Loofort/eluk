package main

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
