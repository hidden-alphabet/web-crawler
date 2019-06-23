package main

import (
	scheduler "github.com/lambda-labs-13-stock-price-2/task-scheduler"
	crawler "github.com/lambda-labs-13-stock-price-2/twitter-advanced-search-crawler-task"
)

/*
  1. Get a query and download the html associated thereof
  2. Create an S3 upload job and a parsing job
  3. The parsing job creates another query job
*/
func main() {
	s := scheduler.NewScheduler(true)

	s.Register("S3Put", crawler.S3PutWorker)
	s.Register("TwitterParse", crawler.TwitterParseWorker)
	s.Register("TwitterSearch", crawler.TwitterSearchWorker)

	job := scheduler.NewJob("TwitterSearch", crawler.TwitterSearchJob{Query: "bitcoin"})

	go s.Start()

	s.Jobs.Push(job)

	<-s.ShouldStop
}
