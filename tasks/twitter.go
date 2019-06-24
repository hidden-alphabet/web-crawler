package tasks

import (
	"bytes"
	"crypto/sha256"
	"github.com/PuerkitoBio/goquery"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"github.com/lambda-labs-13-stock-price-2/task-scheduler"
)

const (
	URL       = "https://twitter.com/search"
	USERAGENT = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) " +
		"AppleWebKit/537.36 (KHTML, like Gecko) " +
		"Chrome/74.0.3729.169 Safari/537.36"
	BUCKET = "hidden-alphabet"
	KEY    = "datasets/webpages/raw/twitter.com"
)

type TwitterSearchJob struct {
	Query       string
	MaxPosition *string
}

type TwitterParseJob struct {
	HTML  []byte
	Query string
}

/*
  Retrieve HTML from twitter.com/search
*/
func TwitterSearchWorker(ctx interface{}) *scheduler.Result {
	output := &scheduler.Result{}

	job, ok := (ctx).(TwitterSearchJob)
	if !ok {
		output.Err = errors.New("Coercion to TwitterSearchJob failed.")
		return output
	}

	req, err := http.NewRequest("GET", URL, nil)
	if err != nil {
		output.Err = err
		return output
	}

	q := req.URL.Query()
	q.Add("q", url.QueryEscape(job.Query))
	q.Add("f", "tweets")
	q.Add("src", "typd")
	q.Add("vertical", "default")

	if job.MaxPosition != nil {
		q.Add("max_position", *job.MaxPosition)
	}

	req.URL.RawQuery = q.Encode()
	req.Header.Set("User-Agent", USERAGENT)

	client := &http.Client{}

	res, err := client.Do(req)
	if err != nil {
		output.Err = err
		return output
	}

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		output.Err = err
		return output
	}

	hash := sha256.Sum256(data)
	key := fmt.Sprintf("%s/%x.html", KEY, hash[:])

	upload := scheduler.NewJob("S3Put", S3PutJob{
		Region: "us-west-2",
		Bucket: BUCKET,
		Key:    key,
		File:   data,
	})

	parse := scheduler.NewJob("TwitterParse", TwitterParseJob{
		HTML:  data,
		Query: job.Query,
	})

	output.Jobs = append(output.Jobs, upload)
	output.Jobs = append(output.Jobs, parse)

	return output
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
