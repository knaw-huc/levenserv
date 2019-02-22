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

	"github.com/gin-gonic/gin"
	"github.com/knaw-huc/levenserv/internal/levenshtein"
	"github.com/knaw-huc/levenserv/internal/vp"
)

type nnIndex struct {
	debug     bool
	metric    string
	normName  string
	normalize func(string) string
	timeout   time.Duration

	*vp.Tree
}

func (i *nnIndex) init(strs <-chan string) (http.Handler, error) {
	m, err := metricByName(i.metric)
	if err != nil {
		return nil, err
	}

	if i.debug {
		log.Print("building index")
	}
	i.Tree, err = vp.New(context.Background(), m, strs)
	if err != nil {
		return nil, err
	}
	if i.debug {
		log.Printf("done, %d words", i.Tree.Len())
	}

	if !i.debug {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()
	r.GET("/info", i.info)
	r.GET("/keys", i.allKeys)
	r.POST("/knn", i.knn)
	return r, nil
}

func metricByName(name string) (m vp.Metric, err error) {
	switch name {
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
func (i *nnIndex) allKeys(c *gin.Context) {
	_, err := c.Writer.Write([]byte("["))
	if err != nil {
		return
	}

	enc := json.NewEncoder(c.Writer)
	n := i.Tree.Len()

	i.Tree.Do(func(key string) bool {
		err := enc.Encode(key)
		if err != nil {
			return false
		}

		if n--; n != 0 {
			_, err = c.Writer.Write([]byte(","))
		}
		return err == nil
	})
}

// info sends some information about the index.
func (i *nnIndex) info(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"metric": i.metric,
		"norm":   i.normName,
		"size":   i.Tree.Len(),
	})
}

func (i *nnIndex) knn(c *gin.Context) {
	params := defaultParams
	err := c.BindJSON(&params)
	switch {
	case params.K < 0:
		err = errors.New("missing or negative k")
	case params.Query == "":
		err = errors.New("missing or empty query string")
	case params.MaxDist < 0:
		err = fmt.Errorf("negative maximum distance %f", params.MaxDist)
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var pred func(string) bool
	if params.Regexp != "" {
		re, err := regexp.Compile(params.Regexp)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		pred = re.MatchString
	}

	ctx, _ := context.WithTimeout(c, i.timeout)
	q := params.Query
	if i.normalize != nil {
		q = i.normalize(q)
	}
	r, err := i.Tree.Search(ctx, q, params.K, params.MaxDist, pred)
	if err != nil {
		status := http.StatusInternalServerError
		if err == context.DeadlineExceeded {
			status = http.StatusRequestTimeout
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, r)
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
