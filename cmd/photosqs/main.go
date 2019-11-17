package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"image/jpeg"
	"mime"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/nfnt/resize"
	log "github.com/sirupsen/logrus"
)

// Usage:
//    go run main.go -n queue_name -t timeout
func main() {

	var queueName string
	var timeout int

	flag.StringVar(&queueName, "n", "GenerateThumbnail", "Queue name")
	flag.IntVar(&timeout, "t", 20, "(Optional) Timeout in seconds for long polling")
	flag.Parse()

	if len(queueName) == 0 {
		flag.PrintDefaults()
		log.Fatal("Queue name required")
	}

	log.Info("Initializing SQS")
	sess := session.Must(session.NewSession())
	svc := sqs.New(sess)

	res, err := svc.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: aws.String(queueName),
	})

	if err != nil {
		log.Fatal("Could not find queue:", err)
		return
	}

	queueURL := aws.StringValue(res.QueueUrl)

	log.Info("Queue created: ", queueURL)

	log.Info("Enabling long polling on queue")

	_, err = svc.SetQueueAttributes(&sqs.SetQueueAttributesInput{
		QueueUrl: aws.String(queueURL),
		Attributes: aws.StringMap(map[string]string{
			"ReceiveMessageWaitTimeSeconds": strconv.Itoa(timeout),
		}),
	})
	if err != nil {
		log.Fatalf("Unable to update queue %q, %v.", queueName, err)
	}

	log.Infof("Successfully updated queue %q.", queueName)

	for {
		log.Println("Start polling SQS")
		res, err := svc.ReceiveMessage(&sqs.ReceiveMessageInput{
			QueueUrl:        aws.String(queueURL),
			WaitTimeSeconds: aws.Int64(int64(timeout))})
		if err != nil {
			log.Println(err)
			continue
		}

		log.Infof("Received %d messages.", len(res.Messages))
		if len(res.Messages) > 0 {
			var wg sync.WaitGroup
			wg.Add(len(res.Messages))
			for i := range res.Messages {
				go func(m *sqs.Message) {
					log.Info("Spawned worker goroutine")
					defer wg.Done()
					if err := handleMessage(svc, &queueURL, res.Messages[0]); err != nil {
						log.Error(err)
					}
				}(res.Messages[i])
			}
			wg.Wait()
		}
	}

}

// A s3EventMsg represents the SQS message provided by S3 Notifications. This
// is an abbreviated form of the message since not all fields are used by this
// service.
type s3EventMsg struct {
	Records []struct {
		S3 struct {
			Bucket struct {
				Name string
			}
			Object struct {
				Key string
			}
		}
	}
}

func handleMessage(svc *sqs.SQS, q *string, m *sqs.Message) error {

	data := aws.StringValue(m.Body)

	log.Info("Message Body:", data)

	var s3msg s3EventMsg

	if err := json.Unmarshal([]byte(data), &s3msg); err != nil {
		log.Error(err)
		return err
	}

	bucket := s3msg.Records[0].S3.Bucket.Name
	key := s3msg.Records[0].S3.Object.Key

	log.Info("Bucket: ", bucket)
	log.Info("Key: ", key)

	if err := generateThumbnail(bucket, key); err != nil {
		return err
	}

	log.Info("Deleting message: ", aws.StringValue(m.MessageId))

	_, err := svc.DeleteMessage(&sqs.DeleteMessageInput{
		QueueUrl:      q,
		ReceiptHandle: m.ReceiptHandle,
	})

	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func generateThumbnail(bucketName, key string) error {
	log.Infof("Fetching s3://%v/%v", bucketName, key)

	sess := session.New()
	buff := &aws.WriteAtBuffer{}
	s3dl := s3manager.NewDownloader(sess)
	_, err := s3dl.Download(buff, &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(key),
	})

	if err != nil {
		log.Fatalf("Could not download from S3: %v", err)
	}

	log.Infof("Decoding image")

	imageBytes := buff.Bytes()
	reader := bytes.NewReader(imageBytes)

	img, err := jpeg.Decode(reader)
	if err != nil {
		log.Fatalf("bad response: %s", err.Error())
		return err
	}

	log.Infof("Generating thumbnail")
	thumbnail := resize.Thumbnail(600, 600, img, resize.Lanczos3)

	log.Infof("Encoding image for upload to S3")
	buf := new(bytes.Buffer)
	err = jpeg.Encode(buf, thumbnail, nil)

	if err != nil {
		log.Errorf("JPEG encoding error: %s", err.Error())
		return err
	}

	// Filename: e5f97749-5d2f-4770-89ce-5d68b1a90f7b/filename.jpg
	// Thumbnail: e5f97749-5d2f-4770-89ce-5d68b1a90f7b/thumb/filename.jpg

	thumbkey := strings.Replace(key, "/", "/thumb/", -1)

	log.Infof("Preparing S3 object: %s", thumbkey)

	uploader := s3manager.NewUploader(sess)
	result, err := uploader.Upload(&s3manager.UploadInput{
		Body:        bytes.NewReader(buf.Bytes()),
		Bucket:      aws.String(bucketName),
		Key:         aws.String(thumbkey),
		ContentType: aws.String(mime.TypeByExtension(filepath.Ext(thumbkey))),
	})

	if err != nil {
		log.Error("Failed to upload", err)
		return err
	}

	log.Println("Successfully uploaded to", result.Location)

	return nil
}
