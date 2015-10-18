package main

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
	Stage   Stage
	Invalid bool
}

func NewShop(link, key string) Shop {
	match := domainRE.FindStringSubmatch(link)
	return Shop{
		Link:  link,
		Host:  match[1],
		Key:   key,
		Lang:  "en",
		Stage: STG_NEW,
	}
}
