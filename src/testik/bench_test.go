package main

import (
	"testing"
	"time"
)

const PARALLEL = 150

func TestFuck(t *testing.T) {
	t.Log("I'm testing")
}

func Benchmark_Sync(b *testing.B) {
	b.SetParallelism(PARALLEL)
	var tm int64
	var cnt int64
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			start := time.Now()
			c, rel := dbsync()
			err := job(c)
			rel()
			if err != nil {
				b.Fatal(err)
			}
			end := time.Now()

			tm += end.Sub(start).Nanoseconds()
			cnt++
		}
	})

	b.Logf("time per operation: %d; ctn: %d", tm/(cnt*1000000), cnt)
}

func Benchmark_Async(b *testing.B) {
	b.SetParallelism(PARALLEL)
	var tm int64
	var cnt int64
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			start := time.Now()
			c, rel := db()
			err := job(c)
			rel()
			if err != nil {
				b.Fatal(err)
			}
			end := time.Now()

			tm += end.Sub(start).Nanoseconds()
			cnt++

		}
	})

	b.Logf("time per operation: %d; ctn: %d", tm/(cnt*1000000), cnt)
}
