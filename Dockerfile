FROM golang:1.12-alpine as build

# Git is needed by go get.
RUN apk add --no-cache git

RUN go get -v -d \
    github.com/julienschmidt/httprouter \
    github.com/stretchr/testify \
    golang.org/x/text

WORKDIR /go/src/github.com/knaw-huc/levenserv
COPY . .

RUN CGO_ENABLED=0 go test ./...
RUN CGO_ENABLED=0 go install -ldflags="-s" .

FROM scratch
COPY LICENSE README.md ./
COPY --from=build /go/bin/levenserv .

ENTRYPOINT ["./levenserv", "-addr", ":8080"]
