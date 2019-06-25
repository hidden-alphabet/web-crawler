FROM golang:alpine

RUN mkdir /app

WORKDIR /app

# h/t https://github.com/docker-library/golang/issues/80#issuecomment-174085707
RUN apk add --no-cache git

RUN go get -u github.com/lambda-labs-13-stock-price-2/web-crawler

RUN git clone https://github.com/lambda-labs-13-stock-price-2/web-crawler.git . && \
    go build main.go 

RUN apk del git

ENTRYPOINT ["./main"]
