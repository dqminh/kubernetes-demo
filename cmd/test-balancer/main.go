package main

import (
	"fmt"
	"net/http"
	"time"
)

var (
	endpoint = "http://172.17.8.102/env"
	interval = 10 * time.Millisecond
	duration = 5 * time.Second
	workers  = 5
)

func main() {
	responses := make(chan *http.Response)
	ticker := time.NewTicker(interval)

	for i := 0; i < workers; i++ {
		go hit(endpoint, ticker.C, responses)
	}

	hits := int(duration / interval)
	results := make([]*http.Response, 0, hits)
	after := time.After(duration)
	for hits > 0 {
		select {
		case <-after:
			ticker.Stop()
			hits = 0
		case r := <-responses:
			results = append(results, r)
			hits--
		}
	}

	goapp := make(map[string]int)
	nginx := make(map[string]int)

	fmt.Printf("got %d responses\n", len(results))
	for _, r := range results {
		goName := r.Header.Get("X-Goserved-By")
		nginxName := r.Header.Get("X-Served-By")
		goapp[goName]++
		nginx[nginxName]++
	}
	fmt.Println(goapp)
	fmt.Println(nginx)
}

func hit(endpoint string, tick <-chan time.Time, responses chan<- *http.Response) {
	for _ = range tick {
		r, err := http.Get(endpoint)
		if err == nil {
			responses <- r
		}
	}
}
