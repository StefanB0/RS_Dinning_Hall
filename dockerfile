# syntax=docker/dockerfile:1
FROM golang:1.19-alpine

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
COPY resources ./resources
# COPY . /app/

RUN go mod download

COPY *.go ./

RUN go build -o /docker-dinning-hall

EXPOSE 8068

CMD [ "/docker-dinning-hall" ]