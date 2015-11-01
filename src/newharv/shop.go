package main

import "net/url"

const (
	STG_NEW int = iota
	STG_CHECKED
	STG_MAIL
	STG_STAT
)

type Shop struct {
	Link    string
	Host    string
	Key     string
	Lang    string
	Stage   int
	Invalid bool
}

func NewShop(link, key string) (Shop, error) {
	sh := Shop{
		Link:  link,
		Key:   key,
		Lang:  "en",
		Stage: STG_NEW,
	}

	url, err := url.Parse(link)
	if err != nil {
		return sh, err
	}

	sh.Host = url.Host
	return sh, err
}
