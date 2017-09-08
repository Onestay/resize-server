package main

import (
	"encoding/json"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
)

func startSQSPollInterval(timeout int64, interval time.Duration) {
	ticker := time.NewTicker(interval)
	quit := make(chan struct{})
	error := make(chan error)
	go func() {
		for {
			select {
			case <-ticker.C:
				poll(timeout, error)
			case <-quit:
				ticker.Stop()
				return
			case err := <-error:
				if err != nil {
					log.Println("An error occured while rezising ", err)
					return
				}
			}
		}
	}()
}

func poll(timeout int64, errChan chan error) {
	log.Println("started polling")
	res, err := svc.ReceiveMessage(&sqs.ReceiveMessageInput{
		QueueUrl:              queueURL,
		AttributeNames:        aws.StringSlice([]string{"SentTimestamp"}),
		MaxNumberOfMessages:   aws.Int64(10),
		MessageAttributeNames: aws.StringSlice([]string{"All"}),
		WaitTimeSeconds:       aws.Int64(timeout),
	})
	if err != nil {
		log.Println()
	}
	log.Println("got message ", len(res.Messages))
	go func() {
		for _, message := range res.Messages {
			mes := sqsS3Message{}

			json.Unmarshal([]byte(*message.Body), &mes)
			log.Println("processing image ", mes.Records[0].S3.Object.Key)
			rez(mes.Records[0].S3.Object.Key, errChan)

		}
	}()

	go func() {
		for i := 0; i < len(res.Messages); i++ {
			_, err := svc.DeleteMessage(&sqs.DeleteMessageInput{
				QueueUrl:      queueURL,
				ReceiptHandle: res.Messages[i].ReceiptHandle,
			})
			if err != nil {
				log.Println("error deleting ", err)
			}
		}
	}()
}
