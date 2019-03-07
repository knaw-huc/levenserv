package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"testing"
	"time"
)

func makeHandler(metric string) http.Handler {
	idx := nnIndex{
		debug:      false,
		metricName: metric,
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
	return h
}

func TestInfo(t *testing.T) {
	h := makeHandler("levenshtein_bytes")

	req := httptest.NewRequest("GET", "/info", nil)

	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	resp := w.Result()

	var m map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&m)

	if !reflect.DeepEqual(m, map[string]interface{}{
		"metric": "levenshtein_bytes",
		"norm":   "nfkd",
		"size":   4.,
	}) {
		t.Errorf("unexpected result %v", m)
	}
}

func TestKnnJaccard(t *testing.T) {
	testKnn(t, "jaccard_trigrams", "brat", 2, []result{
		{"distance": 0.75, "point": "bar"},
		{"distance": 0.8461538461538461, "point": "baz"},
	})
}

func TestKnnLevenshtein(t *testing.T) {
	testKnn(t, "levenshtein", "foobar", 2, []result{
		{"point": "bar", "distance": 3.},
		{"point": "foo", "distance": 3.},
	})
}

// We could decode to []vp.Result, but we'll simulate a client that
// doesn't share the vp package with us.
type result map[string]interface{}

func testKnn(t *testing.T, metric, query string, k int, expect []result) {
	h := makeHandler(metric)

	body, _ := json.Marshal(struct {
		K     int    `json:"k"`
		Query string `json:"query"`
	}{k, query})

	req := httptest.NewRequest("POST", "/knn", bytes.NewReader(body))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	resp := w.Result()

	var results []result
	json.NewDecoder(resp.Body).Decode(&results)
	sortResults(results)

	if !reflect.DeepEqual(results, expect) {
		t.Errorf("unexpected result:\n%vwanted:\n%v", results, expect)
	}
}

// Sort results by distance first, point second.
func sortResults(r []result) {
	sort.Slice(r, func(i, j int) bool {
		return r[i]["distance"].(float64) < r[j]["distance"].(float64) ||
			r[i]["point"].(string) < r[j]["point"].(string)
	})
}
