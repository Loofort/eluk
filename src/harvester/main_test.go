package main

import (
	"testing"
)

func Test_Google(t *testing.T) {
	body := requestG("shop", 3)
	t.Logf("Google Body:\n%s", body)
}

func Test_GLinks(t *testing.T) {
	links := linksFromG("shop", 3)
	t.Logf("Google links count :%d", len(links))
	if len(links) < 9 {
		t.Fatalf("can't obtain links")
	}

}
