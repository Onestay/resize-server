This is just a simple server I wrote for the nfnt/resize package for Hikari.

## For production
The image is on [Dockerhub](https://hub.docker.com/r/onestay/resize-server/). It needs the following env variables to run:

- BUCKET_NAME=The bucket where to get the files from and where to upload them
- AWS_ACCESS_KEY_ID=Your aws access key id
- AWS_SECRET_ACCESS_KEY=Your aws secret access key
- AWS_REGION=the region of the bucket

After that you can hit the endpoint with a query paramter 'key' which is the picture to be resized. It will then produce a thumbnail of max 200x200 and upload it to the same bucket under a folder called /thumb.

## For dev
Fill out all the env vars and docker-compose up 