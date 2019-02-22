package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/text/unicode/norm"
)

func main() {
	var (
		addr   = flag.String("addr", ":8080", "bind to this address")
		debug  = flag.Bool("debug", false, "debugging mode")
		format = flag.String("format", "lines", "input format: lines or json")
		metric = flag.String("metric", "levenshtein",
			"string distance metric to use")
		normalFlag = flag.String("normalize", "",
			"Unicode normalization: NFC, NFD, NFKC, NFKD or empty for none")
		timeout = flag.Int("timeout", 60, "request timeout in seconds")

		err   error
		input = ioutil.NopCloser(os.Stdin)
	)

	flag.Parse()
	switch flag.NArg() {
	case 0:
	case 1:
		if arg := flag.Args()[0]; arg != "-" {
			input, err = os.Open(arg)
			if err != nil {
				log.Fatal(err)
			}
		}
	default:
		flag.Usage()
		os.Exit(1)
	}

	normalize, err := normalForm(*normalFlag)
	if err != nil {
		log.Fatal(err)
	}

	readStrings := readLines
	switch strings.ToLower(*format) {
	case "json":
		readStrings = readJSON
	case "lines":
	default:
		log.Fatalf("unknown input format %q", *format)
	}

	strs := make(chan string, 1)
	go func() {
		defer close(strs)
		defer input.Close()
		readStrings(input, strs)
	}()

	strs = mapFn(strs, normalize)

	t := time.Duration(*timeout) * time.Second
	idx := nnIndex{
		debug:     *debug,
		metric:    *metric,
		normName:  strings.ToLower(*normalFlag),
		normalize: normalize,
		timeout:   t,
	}
	h, err := idx.init(strs)
	if err != nil {
		log.Fatal(err)
	}

	srv := http.Server{
		Addr:         *addr,
		Handler:      h,
		ReadTimeout:  t,
		WriteTimeout: t,
	}
	log.Fatal(srv.ListenAndServe())
}

// normalForm returns a Unicode normalization function.
func normalForm(name string) (nf func(string) string, err error) {
	switch strings.ToLower(name) {
	case "":
	case "nfc":
		nf = norm.NFC.String
	case "nfd":
		nf = norm.NFD.String
	case "nfkc":
		nf = norm.NFKC.String
	case "nfkd":
		nf = norm.NFKD.String
	default:
		err = fmt.Errorf("unknown string normalization %q", name)
	}
	return
}

func mapFn(ch chan string, fn func(string) string) chan string {
	if fn == nil {
		return ch
	}

	out := make(chan string, 1)
	go func() {
		defer close(out)
		for s := range ch {
			out <- fn(s)
		}
	}()

	return out
}

func readLines(r io.Reader, strs chan<- string) {
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		strs <- sc.Text()
	}
	if err := sc.Err(); err != nil {
		log.Fatal(err)
	}
}

func readJSON(r io.Reader, strs chan<- string) {
	dec := json.NewDecoder(os.Stdin)
	for dec.More() {
		var s string
		err := dec.Decode(&s)
		if err != nil {
			log.Fatal(err)
		}
		strs <- s
	}
}
