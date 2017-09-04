package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"image"
	"image/jpeg"
	"io"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/nfnt/resize"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"github.com/buaazp/fasthttprouter"
	"github.com/valyala/fasthttp"
)

type response struct {
	OK  bool   `json:"ok"`
	Err string `json:"err"`
}

var (
	downloader *s3manager.Downloader
	uploader   *s3manager.Uploader
	bucket     string
)

func init() {
	sess, err := session.NewSession()
	if err != nil {
		log.Fatal("Could not create aws session ", err)
	}

	downloader = s3manager.NewDownloader(sess)
	uploader = s3manager.NewUploader(sess)

	bucket = os.Getenv("BUCKET_NAME")
	if len(bucket) == 0 {
		log.Fatal("Couldn't find env var bucket")
	}
}

func main() {
	router := fasthttprouter.New()

	router.POST("/", thumbnail)

	log.Fatal(fasthttp.ListenAndServe(":3001", router.Handler))
}

func thumbnail(ctx *fasthttp.RequestCtx) {
	if !ctx.QueryArgs().Has("key") {
		handleError(ctx, errors.New("No argument \"key\" provided"))
		return
	}
	key := ctx.QueryArgs().Peek("key")
	file, err := os.Create(string(key))
	if err != nil {
		handleError(ctx, err)
		return
	}
	defer os.Remove(string(key))

	downloaded := make(chan bool)
	b := make(chan io.Reader)

	go awsDownload(file, string(key), downloaded, ctx)

	<-downloaded

	go resizeImage(file, b, ctx)

	buffer := <-b

	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String("thumb/" + string(key)),
		Body:        buffer,
		ContentType: aws.String("image/jpeg"),
	})
	if err != nil {
		handleError(ctx, err)
	}

	res := response{true, ""}
	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	jsonErr := json.NewEncoder(ctx).Encode(res)
	if jsonErr != nil {
		log.Fatal("Couldn't marshal json ", jsonErr)
	}

}

func awsDownload(buffer io.WriterAt, key string, c chan bool, ctx *fasthttp.RequestCtx) {
	_, err := downloader.Download(buffer, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		handleError(ctx, err)
		return
	}

	c <- true
}

func resizeImage(file io.Reader, buffer chan io.Reader, ctx *fasthttp.RequestCtx) {
	img, _, err := image.Decode(file)
	if err != nil {
		handleError(ctx, err)
	}

	b := bytes.NewBuffer([]byte(""))
	m := resize.Thumbnail(200, 200, img, resize.Lanczos3)
	jpeg.Encode(b, m, nil)

	buffer <- b
}

func handleError(ctx *fasthttp.RequestCtx, err error) {
	res := response{false, err.Error()}
	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusInternalServerError)
	jsonErr := json.NewEncoder(ctx).Encode(res)
	if jsonErr != nil {
		log.Fatal("Couldn't marshal json ", jsonErr)
	}
}
