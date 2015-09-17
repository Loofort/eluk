package main

import (
	"http"
	"log"
)

func gLinks(url string) []string {
	body := query(gReq(url))

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
	//log.Printf("Got %d links for key %s page %d", len(links), kwd, page)
	return links
}

func query(req *http.Request) string {

	//dump, _ := httputil.DumpRequest(req, false)
	//log.Printf("request to Google:\n%s\n", string(dump))

	resp, err := web.Do(req)
	if err != nil {
		log.Panicf("quering url %s got error %v\n", url, err)
		return ""
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Panicf("can't read body for url %s, got error %v\n", url, err)
		return ""
	}

	s := string(body)
	//log.Printf("Got Google body: %s", s)
	return s
}

func gUrl(kwd string, page int) string {
	return fmt.Sprintf("http://www.google.com/search?q=%s&gbv=2&gws_rd=ssl&num=100&start=%d00", url.QueryEscape(kwd), page)
}

func gReq(url string) *http.Request {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Panicf("error when creating request for '%s': %v\n", kwd, err)
		return nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/44.0.2403.155 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Encoding", "gzip, deflate, sdch")
	req.Header.Set("Accept-Language", "ru-RU,ru;q=0.8,en-US;q=0.6,en;q=0.4,vi;q=0.2")
	req.Header.Set("Cache-Control", "max-age=0")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	return req
}
