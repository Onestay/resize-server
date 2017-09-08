package main

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/sqs"

	"github.com/buaazp/fasthttprouter"
	"github.com/valyala/fasthttp"
)

type response struct {
	OK  bool   `json:"ok"`
	Err string `json:"err"`
}

var (
	svc        *sqs.SQS
	downloader *s3manager.Downloader
	uploader   *s3manager.Uploader
	bucket     string
	queueName  string
	queueURL   *string
)

func init() {
	sess, err := session.NewSession()
	if err != nil {
		log.Fatal("Could not create aws session ", err)
	}

	downloader = s3manager.NewDownloader(sess)
	uploader = s3manager.NewUploader(sess)

	svc = sqs.New(sess)

	queueName = os.Getenv("SQS_QUEUE")
	if len(queueName) == 0 {
		log.Println("No env var 'SQS_QUEUE' found. SQS Polling is disabled")
	} else {
		url, err := svc.GetQueueUrl(&sqs.GetQueueUrlInput{
			QueueName: aws.String(queueName),
		})
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok && aerr.Code() == sqs.ErrCodeQueueDoesNotExist {
				log.Fatal("Unable to find queue ", queueName, err)
			}
			log.Fatal("error getting queue url", err)
		}
		queueURL = url.QueueUrl
	}

	bucket = os.Getenv("BUCKET_NAME")
	if len(bucket) == 0 {
		log.Fatal("Couldn't find env var bucket")
	}
}

func main() {
	router := fasthttprouter.New()
	router.GET("/", thumbnail)
	router.GET("/sqs", startSQSPolling)
	log.Fatal(fasthttp.ListenAndServe(":3001", router.Handler))
}

func thumbnail(ctx *fasthttp.RequestCtx) {
	if !ctx.QueryArgs().Has("key") {
		handleError(ctx, errors.New("No argument \"key\" provided"))
		return
	}

	key := ctx.QueryArgs().Peek("key")
	errorChan := make(chan error)
	go rez(string(key), errorChan)

	err := <-errorChan
	if err != nil {
		handleError(ctx, errors.New("Couldn't resize image"))
		log.Println(err)
		return
	}

	res := response{true, ""}
	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	jsonErr := json.NewEncoder(ctx).Encode(res)
	if jsonErr != nil {
		log.Fatal("Couldn't marshal json ", jsonErr)
	}

}

func startSQSPolling(ctx *fasthttp.RequestCtx) {
	if len(*queueURL) == 0 {
		handleError(ctx, errors.New("SQS hasnt't been setup"))
	}
	log.Println("starting sqs polling interval")
	startSQSPollInterval(20, 30*time.Second)
}
