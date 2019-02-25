package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

var srv *httptest.Server

func init() {
	idx := nnIndex{
		debug:      false,
		metricName: "levenshtein",
		normName:   "nfkd",
		timeout:    2 * time.Second,
	}

	ch := make(chan string)
	go func() {
		defer close(ch)
		for _, s := range []string{"foo", "bar", "baz", "quux"} {
			ch <- s
		}
	}()

	h, err := idx.init(ch)
	if err != nil {
		panic(err)
	}

	srv = httptest.NewServer(h)
}

func TestInfo(t *testing.T) {
	resp, err := http.Get(srv.URL + "/info")
	if err != nil {
		t.Fatal(err)
	}

	var m map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&m)

	if !reflect.DeepEqual(m, map[string]interface{}{
		"metric": "levenshtein",
		"norm":   "nfkd",
		"size":   4.,
	}) {
		t.Errorf("unexpected result %v", m)
	}
}

func TestKnn(t *testing.T) {
	resp, err := http.Post(srv.URL+"/knn", "application/json",
		strings.NewReader(`{"k": 2, "query": "foobar"}`))
	if err != nil {
		t.Fatal(err)
	}

	// We could decode to []vp.Result, but we'll simulate a client that
	// doesn't share the vp package with us.
	var result []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	// Whether "bar" comes before "foo" is indeterminate.
	sort.Slice(result, func(i, j int) bool {
		return result[i]["point"].(string) < result[j]["point"].(string)
	})

	if !reflect.DeepEqual(result, []map[string]interface{}{
		{"point": "bar", "distance": 3.},
		{"point": "foo", "distance": 3.},
	}) {
		t.Errorf("unexpected result %v", result)
	}
}
