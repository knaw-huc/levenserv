FROM golang:1.11.5-alpine as build

# Git is needed by go get.
RUN apk add --no-cache git

RUN go get -v -d github.com/gin-gonic/gin golang.org/x/text

WORKDIR /go/src/github.com/knaw-huc/levenserv
COPY . .

RUN CGO_ENABLED=0 go test ./...
RUN CGO_ENABLED=0 go install .

FROM scratch
COPY --from=build /go/bin/levenserv .
COPY LICENSE .
COPY README.md .

ENTRYPOINT ["./levenserv"]
