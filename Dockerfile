FROM golang:1.27-alpine


RUN go install github.com/air-verse/air@v1.61.7

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

CMD [ "air" ]
