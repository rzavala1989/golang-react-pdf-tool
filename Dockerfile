FROM golang:1.24.1
LABEL author="ricardozavala"

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o pdfserver .

EXPOSE 8080
CMD ["./pdfserver"]
