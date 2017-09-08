package main

import (
	"bytes"
	"encoding/json"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"os"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/nfnt/resize"
	"github.com/valyala/fasthttp"
)

func rez(key string, okay chan error) {

	if !strings.Contains(key, "jpg") && !strings.Contains(key, "jpeg") && !strings.Contains(key, "gif") && !strings.Contains(key, "png") {
		return
	}

	file, err := os.Create(string(key))
	if err != nil {
		okay <- err
		return
	}
	defer os.Remove(string(key))

	err = awsDownload(file, string(key))
	if err != nil {
		okay <- err
		return
	}
	buffer, err := resizeImage(file, key)
	if err != nil {
		okay <- err
		return
	}

	err = awsUpload(key, buffer)
	if err != nil {
		okay <- err
		return
	}

	okay <- nil
}

func awsDownload(buffer io.WriterAt, key string) error {
	_, err := downloader.Download(buffer, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return err
	}

	return nil
}

func resizeImage(file io.Reader, key string) (io.Reader, error) {
	var img image.Image
	var err error
	if path.Ext(key) == ".png" {
		img, err = png.Decode(file)
	} else if path.Ext(key) == ".gif" {
		img, err = gif.Decode(file)
	} else if path.Ext(key) == ".jpg" || path.Ext(key) == ".jpeg" {
		img, err = jpeg.Decode(file)
	}
	if err != nil {
		return nil, err
	}

	b := bytes.NewBuffer([]byte(""))
	m := resize.Thumbnail(200, 200, img, resize.Lanczos3)
	err = jpeg.Encode(b, m, nil)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func awsUpload(key string, buffer io.Reader) error {
	_, err := uploader.Upload(&s3manager.UploadInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String("thumb/" + string(key)),
		Body:        buffer,
		ContentType: aws.String("image/jpeg"),
	})
	if err != nil {
		return err
	}

	return nil
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
