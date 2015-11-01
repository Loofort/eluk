package db

import (
	"golang.org/x/net/context"
)

func Upsert(ctx context.Context, shop interface{}) error {
	return nil
}

func IsDub(err error) bool {
	return false
}
