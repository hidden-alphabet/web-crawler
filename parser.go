package twittercrawler

import (
	"bytes"
	"errors"
	"github.com/PuerkitoBio/goquery"
	"github.com/lambda-labs-13-stock-price-2/task-scheduler"
)

type TwitterParseJob struct {
	HTML  []byte
	Query string
}

func TwitterParseWorker(j interface{}) *scheduler.Result {
	output := &scheduler.Result{}
	position := new(string)

	job, ok := (j).(TwitterParseJob)
	if !ok {
		output.Err = errors.New("Unable to coerce job into TwitterParseJob")
		return output
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(job.HTML))
	if err != nil {
		output.Err = err
		return output
	}

	if p, ok := doc.Find("div.stream-container").Attr("data-max-position"); ok {
		*position = p
	}

	search := scheduler.NewJob("TwitterSearch", TwitterSearchJob{
		Query:       job.Query,
		MaxPosition: position,
	})

	output.Jobs = append(output.Jobs, search)

	return output
}
