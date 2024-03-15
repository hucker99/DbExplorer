FROM golang:1.19.1-alpine

RUN mkdir /app

WORKDIR /app

ADD go.mod .
ADD go.sum .

RUN go mod download
ADD . .

EXPOSE 8000

CMD ["go", "run", "."]