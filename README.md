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
look like other words. To get the ten words most similar to "foobar", do:

    curl -s http://localhost:8080/knn -d '{"query": "foobar", "k": 10}' |
        jq 'map(.point)'

You will a result similar to

    [
      "forbear",
      "isobar",
      "goober",
      "forbad",
      "toolbar",
      "foot",
      "Nicobar",
      "forebear",
      "footwear",
      "footage"
    ]


Usage
-----

The main API endpoint is ``/knn``, which performs a k-nearest neighbor search.
It takes a JSON object with fields ``query`` (string) and ``k`` (integer) and
returns the k strings in Levenserv's index that are closest to the query
string. "Closest" means having the smallest Levenshtein edit distance.

The return value is a list of strings and distances:

    $ curl -s http://localhost:8080/knn -d '{"query": "food", "k": 5}' | jq .
    [
      {
        "distance": 0,
        "point": "food"
      },
      {
        "distance": 1,
        "point": "foods"
      },
      {
        "distance": 1,
        "point": "fond"
      },
      {
        "distance": 1,
        "point": "Wood"
      },
      {
        "distance": 1,
        "point": "rood"
      }
    ]

Results can be filtered by providing a regular expression that they must match,
or a maximum distance, or both:

    $ curl -s http://localhost:8080/knn -d '
        {"query": "food", "k": 5, "maxdist": 1, "regexp": "^f"}' | jq .
    [
      {
        "distance": 0,
        "point": "food"
      },
      {
        "distance": 1,
        "point": "foods"
      },
      {
        "distance": 1,
        "point": "fold"
      },
      {
        "distance": 1,
        "point": "ford"
      },
      {
        "distance": 1,
        "point": "fool"
      }
    ]
