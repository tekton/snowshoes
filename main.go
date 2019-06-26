package main

import (
	"context"
	"encoding/json"
	_ "expvar"
	_ "net/http/pprof"
	"os"
	"log"
	"time"

	"github.com/tekton/snowshoes/lib"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-lambda-go/lambdacontext"
)

// TODO: Validate response string
// TODO: ADD "Actions" support for success and error
// TODO: Base integrations for PagerDuty and Slack (mostly for actions)
// TODO: Validate SSL

func init() {}

func lambdaHandler(ctx context.Context, cwe lib.CloudWatchEvent) {
	lc, _ := lambdacontext.FromContext(ctx)
	log.Println(lc.AwsRequestID)
	log.Println("lc", lc)
	log.Println("cwe", cwe)
	

	var startTime = time.Now()
	var s3config lib.S3Config
	jErr := json.Unmarshal(cwe.Detail, &s3config)
	if jErr != nil {
		log.Println("ERROR IN s3config UNMARSHAL")
		log.Println(jErr)
		os.Exit(1)
	}
	log.Print("s3config: ", s3config)
	lib.Timing("s3config", startTime)
	
	startTime = time.Now() // reset...
	sm := lib.GetServerMapFile(&s3config)
	log.Print("ServerMap", sm)
	lib.Timing("ServerMap", startTime)

	startTime = time.Now() // reset...
	lib.ProcessServerMap(sm)
	log.Println("ServerMap processed!")
	lib.Timing("ServerMap processing", startTime)
	return
}

func main() {
	log.Print("Starting lambda handler...")
	lambda.Start(lambdaHandler)
}
