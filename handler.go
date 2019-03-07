package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"regexp"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/knaw-huc/levenserv/internal/levenshtein"
	"github.com/knaw-huc/levenserv/internal/trigrams"
	"github.com/knaw-huc/levenserv/internal/vp"
)

type nnIndex struct {
	debug      bool
	metricName string
	metric     vp.Metric
	normName   string
	normalize  func(string) string
	timeout    time.Duration

	*vp.Tree
}

func (i *nnIndex) init(strs <-chan string) (h http.Handler, err error) {
	i.metric, err = metricByName(i.metricName)
	if err != nil {
		return
	}

	if i.debug {
		log.Print("building index")
	}
	i.Tree, err = vp.New(context.Background(), i.metric, strs)
	if err != nil {
		return
	}
	if i.debug {
		log.Printf("done, %d words", i.Tree.Len())
	}

	r := httprouter.New()
	r.POST("/distance", i.distance)
	r.GET("/info", i.info)
	r.GET("/keys", i.allKeys)
	r.POST("/knn", i.knn)
	return r, nil
}

func metricByName(name string) (m vp.Metric, err error) {
	switch name {
	case "jaccard_trigrams":
		m = trigrams.JaccardDistanceStrings
	case "levenshtein":
		m = func(a, b string) float64 {
			return float64(levenshtein.DistanceCodepoints(a, b))
		}
	case "levenshtein_bytes":
		m = func(a, b string) float64 {
			return float64(levenshtein.DistanceBytes(a, b))
		}
	default:
		err = fmt.Errorf("unknown metric %q", name)
	}
	return
}

// allKeys sends a JSON representation of the set of keys in i.Tree,
// in some unspecified order.
func (i *nnIndex) allKeys(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	_, err := w.Write([]byte("["))
	if err != nil {
		return
	}

	enc := json.NewEncoder(w)
	n := i.Tree.Len()

	i.Tree.Do(func(key string) bool {
		err := enc.Encode(key)
		if err != nil {
			return false
		}

		if n--; n != 0 {
			_, err = w.Write([]byte(","))
		}
		return err == nil
	})
}

// distance computes the distance between a pair of input strings,
// without considering the indexed strings.
func (i *nnIndex) distance(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var strs [2]string
	err := json.NewDecoder(r.Body).Decode(&strs)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if i.normalize != nil {
		strs[0] = i.normalize(strs[0])
		strs[1] = i.normalize(strs[1])
	}

	d := i.metric(strs[0], strs[1])
	json.NewEncoder(w).Encode(struct {
		M string  `json:"metric"`
		D float64 `json:"distance"`
	}{
		i.metricName, d,
	})
}

// info sends some information about the index.
func (i *nnIndex) info(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	json.NewEncoder(w).Encode(map[string]interface{}{
		"metric": i.metricName,
		"norm":   i.normName,
		"size":   i.Tree.Len(),
	})
}

func (i *nnIndex) knn(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	params := defaultParams
	err := json.NewDecoder(r.Body).Decode(&params)
	switch {
	case params.K < 0:
		err = errors.New("missing or negative k")
	case params.Query == "":
		err = errors.New("missing or empty query string")
	case params.MaxDist < 0:
		err = fmt.Errorf("negative maximum distance %f", params.MaxDist)
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	var pred func(string) bool
	if params.Regexp != "" {
		re, err := regexp.Compile(params.Regexp)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		pred = re.MatchString
	}

	ctx, _ := context.WithTimeout(r.Context(), i.timeout)
	q := params.Query
	if i.normalize != nil {
		q = i.normalize(q)
	}
	result, err := i.Tree.Search(ctx, q, params.K, params.MaxDist, pred)
	if err != nil {
		status := http.StatusInternalServerError
		if err == context.DeadlineExceeded {
			status = http.StatusRequestTimeout
		}
		writeError(w, status, err)
		return
	}

	json.NewEncoder(w).Encode(result)
}

func writeError(w http.ResponseWriter, status int, err error) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(struct {
		Error string `json:"error"`
	}{
		err.Error(),
	})
}

type knnParams struct {
	K       int     "json:`k`"
	MaxDist float64 "json:`maxdist`"
	Query   string  "json:`query`"
	Regexp  string  "json:`regexp`"
}

var defaultParams = knnParams{
	K:       -1,           // must be set by caller
	MaxDist: math.Inf(+1), // find everything
}
