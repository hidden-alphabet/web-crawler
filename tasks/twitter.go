package tasks

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/go-redis/redis"
	"github.com/lambda-labs-13-stock-price-2/task-scheduler"
	"io/ioutil"
	"net/http"
	"net/url"
)

const (
	URL       = "https://twitter.com/search"
	USERAGENT = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) " +
		"AppleWebKit/537.36 (KHTML, like Gecko) " +
		"Chrome/74.0.3729.169 Safari/537.36"
)

type WebCrawler struct {
	Redis  *redis.Client
	Bucket string
	Key    string
}

type Query struct {
	Text string
}

type TwitterSearchJob struct {
	Query       *Query
	MaxPosition *string
}

type TwitterParseJob struct {
	HTML  []byte
	Query *Query
}

func (w *WebCrawler) TwitterSearchWorker(ctx interface{}) *scheduler.Result {
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
	q.Add("q", url.QueryEscape(job.Query.Text))
	q.Add("f", "tweets")
	q.Add("src", "typd")
	q.Add("vertical", "default")

	if job.MaxPosition == nil {
		maxPositionHasBeenSet, err := w.Redis.HExists(job.Query.Text, "max_position").Result()
		if err != nil {
			output.Err = err
			return output
		}

		if maxPositionHasBeenSet {
			max_position, err := w.Redis.HGet(job.Query.Text, "max_position").Result()
			if err != nil {
				output.Err = err
				return output
			}

			job.MaxPosition = &max_position
		}
	}

	q.Add("max_position", *job.MaxPosition)

	req.URL.RawQuery = q.Encode()
	req.Header.Set("User-Agent", USERAGENT)

	client := &http.Client{}

	res, err := client.Do(req)
	if err != nil {
		output.Err = err
		return output
	}

	w.Redis.HSet(job.Query.Text, "max_position", *job.MaxPosition)

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		output.Err = err
		return output
	}

	hash := sha256.Sum256(data)
	key := fmt.Sprintf("%s/%x.html", w.Key, hash[:])

	upload := scheduler.NewJob("S3Put", S3PutJob{
		Region: "us-west-2",
		Bucket: w.Bucket,
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

func (w *WebCrawler) TwitterParseWorker(j interface{}) *scheduler.Result {
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
