Levenserv, the fuzzy string matching service
============================================

Levenserv is a web service that provides nearest neighbor searching in string
collections.


Getting started
---------------

To start using Levenserv, make sure you have a collection of strings ready.
Most Linux systems have a mildly interesting one in ``/usr/share/dict/words``.
Build and run Levenserv with Docker:

    container=$(docker build -q .)
    words=/usr/share/dict/words
    docker run -i -p 8080:8080 $container -debug < $words

You now have a REST API serving port 8080 that you can query for words that
look like other words. To get the three words most similar to "foobar", do:

    curl -s http://localhost:8080/knn -d '{"query": "foobar", "k": 3}' |
        jq .

You will get a result similar to

    [
      {
        "distance": 2,
        "point": "forbad"
      },
      {
        "distance": 2,
        "point": "forbear"
      },
      {
        "distance": 2,
        "point": "isobar"
      }
    ]


Usage
-----

The main API endpoint is ``/knn``, which performs a k-nearest neighbor search.
It takes a JSON object with fields ``query`` (string) and ``k`` (integer) and
returns the k strings in Levenserv's index that are closest to the query
string. By default, "closest" means having the smallest Levenshtein edit
distance.

The return value is a list of strings and distances:

    $ curl -s http://localhost:8080/knn -d '{"query": "foods", "k": 15}' |
        jq -c '.[]'
    {"distance":0,"point":"foods"}
    {"distance":1,"point":"floods"}
    {"distance":1,"point":"fools"}
    {"distance":1,"point":"food's"}
    {"distance":1,"point":"woods"}
    {"distance":1,"point":"moods"}
    {"distance":1,"point":"food"}
    {"distance":1,"point":"foots"}
    {"distance":1,"point":"folds"}
    {"distance":1,"point":"roods"}
    {"distance":1,"point":"goods"}
    {"distance":1,"point":"fords"}
    {"distance":1,"point":"Woods"}
    {"distance":1,"point":"hoods"}
    {"distance":2,"point":"foot"}

Results can be filtered by providing a regular expression that they must match,
or a maximum distance, or both:

    $ curl -s http://localhost:8080/knn -d '
        {"query": "food", "k": 5, "maxdist": 1, "regexp": "^f"}' |
        jq -c '.[]'
    {"distance":0,"point":"food"}
    {"distance":1,"point":"foods"}
    {"distance":1,"point":"fold"}
    {"distance":1,"point":"ford"}
    {"distance":1,"point":"fool"}


Distance metrics
----------------

Aside from Levenshtein distance, Levenserv supports the Jaccard distance on
the sets of trigrams extracted from a pair of strings. Start Levenserv with

    levenserv -metric jaccard_trigrams

to get this distance. Its value is always between zero and one:

    $ curl -XPOST http://localhost:8080/knn -d '{"k": 5, "query": "hello"}' |
        jq -c '.[]'
    {"distance":0,"point":"hello"}
    {"distance":0.21428571428571427,"point":"hellos"}
    {"distance":0.2727272727272727,"point":"hell"}
    {"distance":0.35294117647058826,"point":"Othello"}
    {"distance":0.35294117647058826,"point":"hello's"}

The metric `levenshtein_damerau` gives a version of Levenshtein distance
where a transposition (swap) of two adjacent characters is counted as one
edit operation.


Usage from scripts, without Docker
----------------------------------

See example.sh for a shell script that starts and stops Levenserv, without
using Docker. You need to go install the package first for this to work.
