package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/text/unicode/norm"
)

func main() {
	var (
		addrparam = flag.String("addr", "",
			"bind to this address (default: localhost with random port)")
		debug  = flag.Bool("debug", false, "send debugging ouput to stderr")
		format = flag.String("format", "lines", "input format: lines or json")
		metric = flag.String("metric", "levenshtein",
			"string distance metric to use")
		normalFlag = flag.String("normalize", "",
			"Unicode normalization: NFC, NFD, NFKC, NFKD or empty for none")
		timeout = flag.Int("timeout", 60, "request timeout in seconds")

		err   error
		input = os.Stdin
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
			defer input.Close()
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

	if *debug {
		log.Printf("reading strings from %s", input.Name())
	}
	strs, err := readStrings(input)
	if err != nil {
		log.Fatal(err)
	}

	if normalize != nil {
		for i := range strs {
			strs[i] = normalize(strs[i])
		}
	}

	t := time.Duration(*timeout) * time.Second
	idx := nnIndex{
		debug:      *debug,
		metricName: *metric,
		normName:   strings.ToLower(*normalFlag),
		normalize:  normalize,
		timeout:    t,
	}
	h, err := idx.init(strs)
	if err != nil {
		log.Fatal(err)
	}

	addr := *addrparam
	if addr == "" {
		addr = "localhost:"
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}

	srv := http.Server{
		Addr:         addr,
		Handler:      h,
		ReadTimeout:  t,
		WriteTimeout: t,
	}

	if *addrparam == "" {
		fmt.Printf("http://%s\n", ln.Addr())
	}
	os.Stdout.Close()

	log.Fatal(srv.Serve(ln))
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

func readLines(r io.Reader) (strs []string, err error) {
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		strs = append(strs, sc.Text())
	}
	return strs, sc.Err()
}

func readJSON(r io.Reader) (strs []string, err error) {
	dec := json.NewDecoder(os.Stdin)
	for dec.More() {
		var s string
		err = dec.Decode(&s)
		if err != nil {
			break
		}
		strs = append(strs, s)
	}
	return
}
