package main

import (
	scheduler "github.com/lambda-labs-13-stock-price-2/task-scheduler"
	tasks "github.com/lambda-labs-13-stock-price-2/twitter-advanced-search-crawler-task/tasks"
)

/*
  This should receive requests to get tweets in the form of twitter search requests. For example, 
  "bitcoin since:2019-06-01" is a valid request, whereas "?q=bitcoin&f=tweets" is not. The difference
  of course being that this crawler will receive the search query in the format you would input into
  the twitter.com/search webpage and construct the appropriate HTTP request from it. Once we've received
  a search term, we send it to the TwitterSearchWorker, whose job it is to build that aforementioned
  request. However, before we do so, we want to ensure we're not duplicating work, for this reason we
  need a map which we update with the most recent max_position pagination id given for an arbitrary search
  term. Thus, in the case that we stop and restart work, we can start from where we left off and not have to
  redo all the work previously done. To this end every time we parse a new HTML page we create a job to push 
  the most recently pagination id associated to a search term. 

  - listen for a search request 
  - create a search job
  - receive a search job
  - 
*/
func main() {
	s := scheduler.NewScheduler(true)

	s.Register("S3Put", tasks.S3PutWorker)
  s.Register("RedisPush", tasks.RedisPush)
	s.Register("TwitterParse", tasks.TwitterParseWorker)
	s.Register("TwitterSearch", tasks.TwitterSearchWorker)

	job := scheduler.NewJob("TwitterSearch", crawler.TwitterSearchJob{Query: "bitcoin"})

	go s.Start()

	s.Jobs.Push(job)

	<-s.ShouldStop
}
