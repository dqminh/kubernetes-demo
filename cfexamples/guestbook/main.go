package main

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/xyproto/simpleredis"
)

var pool *simpleredis.ConnectionPool

func EnvHandler(rw http.ResponseWriter, req *http.Request) {
	environment := make(map[string]string)
	for _, item := range os.Environ() {
		splits := strings.Split(item, "=")
		key := splits[0]
		val := strings.Join(splits[1:], "=")
		environment[key] = val
	}

	envJSON := HandleError(json.MarshalIndent(environment, "", "  ")).([]byte)
	rw.Write(envJSON)
}

func HandleError(result interface{}, err error) (r interface{}) {
	if err != nil {
		panic(err)
	}
	return result
}

func main() {
	var hostname string
	var err error
	hostname, err = os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	pool = simpleredis.NewConnectionPoolHost("redis-master:6379")
	defer pool.Close()

	http.Handle("/env", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-GoServed-By", hostname)
		EnvHandler(w, r)
	}))

	http.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-GoServed-By", hostname)
		w.Write(HandleError(pool.Get(0).Do("INFO")).([]byte))
	}))

	http.ListenAndServe(":3000", nil)
}
