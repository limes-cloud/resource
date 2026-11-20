package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Fprintln(os.Stderr, "usage: icon-restore <source-bucket> <target-bucket> <key>...")
		os.Exit(2)
	}

	endpoint := os.Getenv("S3_ENDPOINT")
	accessKey := os.Getenv("S3_AK")
	secretKey := os.Getenv("S3_SK")
	if endpoint == "" || accessKey == "" || secretKey == "" {
		fmt.Fprintln(os.Stderr, "S3_ENDPOINT, S3_AK and S3_SK are required")
		os.Exit(2)
	}

	client := awss3.New(awss3.Options{
		BaseEndpoint: aws.String(endpoint),
		Region:       "us-east-1",
		Credentials:  credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		UsePathStyle: true,
	})
	ctx := context.Background()
	sourceBucket := os.Args[1]
	targetBucket := os.Args[2]

	for _, key := range os.Args[3:] {
		_, err := client.CopyObject(ctx, &awss3.CopyObjectInput{
			Bucket:     aws.String(targetBucket),
			Key:        aws.String(key),
			CopySource: aws.String(sourceBucket + "/" + key),
		})
		if err != nil {
			fmt.Printf("copy %s: ERROR %v\n", key, err)
			continue
		}
		if _, err := client.HeadObject(ctx, &awss3.HeadObjectInput{
			Bucket: aws.String(targetBucket),
			Key:    aws.String(key),
		}); err != nil {
			fmt.Printf("copy %s: verify ERROR %v\n", key, err)
			continue
		}
		fmt.Printf("copy %s: OK\n", key)
	}
}
