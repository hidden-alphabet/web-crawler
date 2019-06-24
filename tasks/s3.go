package task

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/lambda-labs-13-stock-price-2/task-scheduler"
)

type S3PutJob struct {
	Region string
	Bucket string
	Key    string
	File   []byte
}

func S3PutWorker(j interface{}) *scheduler.Result {

	output := &scheduler.Result{}

	if job, ok := (j).(S3PutJob); !ok {
		output.Err = errors.New("Coercion to S3PutJob failed.")
		return output
	} else {
		sess := session.Must(session.NewSession(&aws.Config{Region: aws.String(job.Region)}))
		uploader := s3manager.NewUploader(sess)

		result, err := uploader.Upload(&s3manager.UploadInput{
			Bucket: aws.String(job.Bucket),
			Key:    aws.String(job.Key),
			Body:   bytes.NewReader(job.File),
		})

		if err != nil {
			output.Err = errors.New(fmt.Sprintf("Upload failed: %v", err))
			return output
		}

		fmt.Printf("Successfully uploaded to %s\n", result.Location)

		return output
	}
}
