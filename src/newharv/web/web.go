package web

import (
	"golang.org/x/net/context"
	"golang.org/x/net/context/ctxhttp"
	"io/ioutil"
	"net/http"

	"github.com/golang/glog"
	"net/http/httputil"

	// packages to handle metrics
	"fmt"
	"os"
	"os/signal"
	"time"
)

var HTTP_METRICS_FILE = "metrics.csv"
var MPOOL_SIZE = 1000
var metricc = make(chan metric, MPOOL_SIZE)
var mpanicBorder = MPOOL_SIZE - 1

// we want to measure metrics every second:
//  - average latency at each second
//  - thgroughput Mb/sec
//  - amount of requests in connecting and reading state
//  - amount of success and fail transaction
//
// So we start the ticker that will print info every second into the file
func init() {
	go runMetrics()
}

type metric struct {
	latency int64
	bytes   int
	success bool
}

// runMetrics pushes metrics to file every second
func runMetrics() {
	// open file to write metrics
	file, err := os.Create(HTTP_METRICS_FILE)
	if err != nil {
		panic(err)
	}
	fmt.Fprintln(file, "Latency,Throughput,Sucess,Fail,Total")

	// create channel to catch termination signal
	termc := make(chan os.Signal, 10)
	signal.Notify(termc, os.Interrupt, os.Kill)

	// start ticker
	tickc := time.Tick(1 * time.Second)

	var succ, fail int64
	var latency int64
	var throghput int

	for {
		select {
		// close file if program is going to terminate
		case <-termc:
			file.Close()
			//return
			os.Exit(1)

		// obtain metric
		case m := <-metricc:
			if len(metricc) >= mpanicBorder {
				panic("metrics are slowdown the http speed")
			}

			if !m.success {
				fail++
				continue
			}
			succ++
			latency += m.latency
			throghput += m.bytes

		// write metrics
		case <-tickc:
			var avg int64
			if succ != 0 {
				avg = latency / succ
			}

			fmt.Fprintf(file, "%d,%d,%d,%d,%d\n", avg, throghput, succ, fail, succ+fail)
			succ, fail = 0, 0
			latency, throghput = 0, 0
		}

	}
}

func Get(ctx context.Context, url string) (int, []byte, error) {
	req, err := GetReq(url)
	if err != nil {
		return 0, nil, err
	}

	return Query(ctx, req)
}

func GetReq(url string) (*http.Request, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/44.0.2403.155 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Encoding", "gzip, deflate, sdch")
	req.Header.Set("Accept-Language", "ru-RU,ru;q=0.8,en-US;q=0.6,en;q=0.4,vi;q=0.2")
	req.Header.Set("Cache-Control", "max-age=0")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	return req, nil
}

func Query(ctx context.Context, req *http.Request) (int, []byte, error) {
	m := metric{}
	defer func(m *metric) {
		metricc <- *m
	}(&m)

	if glog.V(10) {
		dump, _ := httputil.DumpRequest(req, true)
		glog.Infoln(string(dump))
	}

	start := time.Now()
	resp, err := ctxhttp.Do(ctx, nil, req)
	m.latency = time.Since(start).Nanoseconds()
	if err != nil {
		return 0, nil, err
	}

	if glog.V(10) {
		dump, _ := httputil.DumpResponse(resp, true)
		glog.Infoln(string(dump))
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}

	m.bytes = len(body)
	m.success = true
	return resp.StatusCode, body, nil
}
