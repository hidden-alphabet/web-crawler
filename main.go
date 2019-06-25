package main

import (
	"./tasks" // "github.com/lambda-labs-13-stock-price-2/web-crawler/tasks"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis"
	"github.com/lambda-labs-13-stock-price-2/task-scheduler"
	"io/ioutil"
	"net/http"
	"os"
)

const (
	AWS_SNS_USER_AGENT = "Amazon Simple Notification Service Agent"
	BUCKET             = "hidden-alphabet"
	KEY                = "datasets/webpages/raw/twitter.com"
)

type SNSNotification struct {
	Type             string `json:"Type"`
	MessageId        string `json:"MessageId"`
	Token            string `json:"Token"`
	TopicArn         string `json:"TopicArn"`
	Message          string `json:"Message"`
	SubscribeURL     string `json:"SubscribeURL"`
	Timestamp        string `json:"Timestamp"`
	SignatureVersion string `json:"SignatureVersion"`
	Signature        string `json:"Signature"`
	SigningCertURL   string `json:"SigningCertURL"`
}

type Request struct {
	Query string `json:"query"`
}

type Server struct {
	Scheduler *scheduler.Scheduler
}

func (s *Server) UnwrapSNSNotification(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.Header.Get("User-Agent") == AWS_SNS_USER_AGENT {
			notification := new(SNSNotification)
			json.NewDecoder(r.Body).Decode(notification)

			switch notification.Type {
			case "SubscriptionConfirmation":
				_, err := http.Get(notification.SubscribeURL)
				if err != nil {
					fmt.Println(err)
				}
			default:
				data := []byte(notification.Message)
				reader := bytes.NewReader(data)
				r.Body = ioutil.NopCloser(reader)
			}
		}

		h(w, r)
	}
}

func (s *Server) HandleSearchRequest(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		request := new(Request)
		json.NewDecoder(r.Body).Decode(request)

		q := &tasks.Query{Text: request.Query}
		job := scheduler.NewJob("TwitterSearch", tasks.TwitterSearchJob{Query: q})

		s.Scheduler.Jobs.Push(job)
	}
}

func main() {
	REDIS_HOST := os.Getenv("REDIS_HOST")
	REDIS_PORT := os.Getenv("REDIS_PORT")

	r := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", REDIS_HOST, REDIS_PORT),
		Password: "",
		DB:       0,
	})

	_, err := r.Ping().Result()
	if err != nil {
		panic(err)
	}

	w := tasks.WebCrawler{Redis: r, Bucket: BUCKET, Key: KEY}
	s := scheduler.NewScheduler(true)

	s.Register("S3Put", w.S3PutWorker)
	s.Register("TwitterParse", w.TwitterParseWorker)
	s.Register("TwitterSearch", w.TwitterSearchWorker)

	go s.Start()

	server := &Server{s}

	handler := server.UnwrapSNSNotification(server.HandleSearchRequest)
	http.HandleFunc("/", handler)
	http.ListenAndServe(":", nil)

	<-s.ShouldStop
}
