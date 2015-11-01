package web

import (
	"bytes"
	"fmt"
	"golang.org/x/net/context"
	"golang.org/x/net/html"
	"regexp"
)

var basketRE = regexp.MustCompile("(?i)\b(?:bag|cart|wishlist|basket|корзина)\b")

func Links(ctx context.Context, url string) ([]string, error) {
	body, err := get(ctx, url)
	if err != nil {
		return nil, err
	}
	return extractLinks(body), nil
}

func get(ctx context.Context, url string) ([]byte, error) {
	code, body, err := Get(ctx, url)
	if err != nil {
		return nil, err
	}

	if code != 200 {
		return nil, fmt.Errorf("http code is %d for page %s", code, url)
	}

	return body, nil
}

func extractLinks(body []byte) []string {
	links := make([]string, 0)
	r := bytes.NewReader(body)

	page := html.NewTokenizer(r)
	for {
		tokenType := page.Next()
		if tokenType == html.ErrorToken {
			return links
		}
		token := page.Token()
		if tokenType == html.StartTagToken && token.DataAtom.String() == "a" {
			for _, attr := range token.Attr {
				if attr.Key == "href" {
					links = append(links, attr.Val)
				}
			}
		}
	}
}

func IsShop(ctx context.Context, url string) (bool, error) {
	body, err := get(ctx, url)
	if err != nil {
		return false, err
	}

	return basketRE.Match(body), nil
}
