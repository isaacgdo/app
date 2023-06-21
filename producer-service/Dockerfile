FROM golang:1.19

WORKDIR /go/src
ENV PATH="/go/bin:${PATH}"
ENV GO111MODULE=on

COPY go.mod .
COPY go.sum .

RUN go mod download && go mod verify

COPY . .

RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -a -v -o producer-service

ENTRYPOINT ["/go/src/producer-service"]