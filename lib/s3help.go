package lib

import (
	_ "expvar"
	"fmt"
	"io/ioutil"
	_ "net/http/pprof"
	"os"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

func GetServerMapFileFromS3(s3Config *S3Config) []byte {
	defer Timing("getServerMapFileFromS3", time.Now())
	log.Println("getServerMapFileFromS3...")
	client, _ := session.NewSession(&aws.Config{
		Region: aws.String(s3Config.Region)},
	)

	s3svc := s3.New(client)
	
	// get meta data -- leaving this in here to remind myself it doesn't help
	// start := time.Now()
	// headObj := &s3.HeadObjectInput{
	// 	Bucket: aws.String(s3Config.Bucket),
	// 	Key:    aws.String(fmt.Sprintf("%s/%s", s3Config.Prefix, s3Config.ServerMap)),
	// }
	// head, err := s3svc.HeadObject(headObj)
	// if err != nil {
	// 	log.Println(err)
	// 	os.Exit(1)
	// }
	// log.Println("Head ", head)
	// Timing("get s3 head: ", start)

	// get the actual file!
	s3GetInfo := &s3.GetObjectInput{
		Bucket: aws.String(s3Config.Bucket),
		Key:    aws.String(fmt.Sprintf("%s/%s", s3Config.Prefix, s3Config.ServerMap)),
	}

	res, err := s3svc.GetObject(s3GetInfo)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	log.Println(res)
	body, bodyErr := ioutil.ReadAll(res.Body)
	if bodyErr != nil {
		log.Println(err)
	}
	log.Println("...getServerMapFileFromS3")
	return body
}