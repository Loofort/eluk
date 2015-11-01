package web

import (
	"fmt"
	"golang.org/x/net/context"
	"net/url"
	"regexp"
)

type GBan struct {
	url string
}

func (gb GBan) Error() string {
	return "google ban, check in browser " + gb.url
}

var glinksRE = regexp.MustCompile(`<a href="/url\?q=([^&]+)`)

// GLinks requests google.com and returns links
// page is from range 1..10 , 100 links per page.
// it is caller responsibility to check if the returned page is the last in serp.
func GLinks(ctx context.Context, kwd string, page int) ([]string, error) {
	gurl := fmt.Sprintf("http://www.google.com/search?q=%s&gbv=2&gws_rd=ssl&num=100&start=%d00", url.QueryEscape(kwd), page)
	req, err := GetReq(gurl)
	if err != nil {
		// error indicates that gurl is not http link, that mean somethink wrong with kwd
		return nil, fmt.Errorf("can't process keyword %s: %v ", kwd, err)
	}

	code, body, err := Query(ctx, req)
	if err != nil {
		return nil, err
	}

	if code == 503 {
		return nil, GBan{gurl}
	}

	match := glinksRE.FindAllStringSubmatch(string(body), -1)
	links := make([]string, 0, len(match))
	for _, m := range match {
		link, err := url.QueryUnescape(m[1])
		if err != nil {
			// error indicate that we probably has a problem with glinksRE and it should be changed
			return nil, fmt.Errorf("can't unescape url %s ; gurl=%s; linksRE = %s", m[1], gurl, glinksRE)
		}

		links = append(links, link)
	}

	return links, nil
}
