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

func TestKnnLevenshtein(t *testing.T) {
	h := makeHandler("levenshtein")

	req := httptest.NewRequest("POST", "/knn",
		strings.NewReader(`{"k": 2, "query": "foobar"}`))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	resp := w.Result()

	// We could decode to []vp.Result, but we'll simulate a client that
	// doesn't share the vp package with us.
	var result []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	// Whether "bar" comes before "foo" is indeterminate.
	sort.Slice(result, func(i, j int) bool {
		x, y := result[i], result[j]
		return x["distance"].(float64) < y["distance"].(float64) ||
			x["point"].(string) < y["point"].(string)
	})

	if !reflect.DeepEqual(result, []map[string]interface{}{
		{"point": "bar", "distance": 3.},
		{"point": "foo", "distance": 3.},
	}) {
		t.Errorf("unexpected result %v", result)
	}
}
