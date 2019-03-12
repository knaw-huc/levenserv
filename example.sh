#!/bin/sh
#
# This shell script exemplifies how to use a temporary Levenserv for an
# ad-hoc collection of words. It assumes levenserv is in your $PATH.

set -e

dictionary=/usr/share/dict/words

# We make a FIFO file (named pipe) at a temporary location. Levenserv will
# use a standard port number by default, and prints its URL to standard
# output.
tempdir=$(mktemp -d)
urlpipe="$tempdir/url"
mkfifo "$urlpipe"

# Install cleanup code.
trap "rm -rf $tempdir" EXIT

# Start Levenserv in the background and install a trap that kills it
# when the shell finishes.
levenserv < "$dictionary" > "$urlpipe" &
trap "kill -HUP $!" EXIT

read url < "$urlpipe"

# The input to Levenserv is hardcoded here. If it's variable, make sure
# it's proper JSON.
curl -s -XPOST "$url/knn" -d '{"query": "mystring", "k": 10}'
