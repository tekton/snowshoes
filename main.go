package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	_ "expvar"
	"fmt"
	"io/ioutil"
	"net/http"
	_ "net/http/pprof"
	"os"
	"sync"
	"log"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type ReqRtn struct {
	Code int    `json:"code"`
	Type string `json:"type"`
	Val  string `json:"val"`
}

type ServerMap []struct {
	DomainTypeId int               `json:"domain_type_id"`
	Val          string            `json:"val"`
	Rtn          ReqRtn            `json:"rtn"`
	ClientID     int               `json:"client_id"`
	URLPath      string            `json:"url_path"`
	Qs           map[string]string `json:"qs,omitempty"`
	Grouping     string            `json:"grouping"`
	DomainName   string            `json:"domain_name"`
}

type S3Config struct {
	Bucket    string
	Prefix    string
	ServerMap string
	Region    string
}

type CloudWatchEvent struct {
	Version    string          `json:"version"`
	ID         string          `json:"id"`
	DetailType string          `json:"detail-type"`
	Source     string          `json:"source"`
	AccountID  string          `json:"account"`
	Time       time.Time       `json:"time"`
	Region     string          `json:"region"`
	Resources  []string        `json:"resources"`
	Detail     json.RawMessage `json:"detail"`
}

// func profileTime(s string) (string, time.Time) {
//     return s, time.Now()
// }

func timing(s string, startTime time.Time) {
    endTime := time.Now()
    log.Println(s, "took", endTime.Sub(startTime))
}

func GrabURLData(url string) *http.Response {
	fmt.Println("GrabURLData", url)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{},
	}
	client := &http.Client{
		Transport: tr,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	res, err := client.Get(url)
	if err != nil {
		log.Print("GET ERROR", err)
		return nil
	}
	return res
}

// TODO: Validate response string
// TODO: ADD "Actions" support for success and error
// TODO: Base integrations for PagerDuty and Slack (mostly for actions)
// TODO: Validate SSL

func getServerMapFileFromS3(s3Config *S3Config) []byte {
	defer timing("getServerMapFileFromS3", time.Now())
	fmt.Println("getServerMapFileFromS3...")
	client, _ := session.NewSession(&aws.Config{
		Region: aws.String(s3Config.Region)},
	)

	s3svc := s3.New(client)

	s3GetInfo := &s3.GetObjectInput{
		Bucket: aws.String(s3Config.Bucket),
		Key:    aws.String(fmt.Sprintf("%s/%s", s3Config.Prefix, s3Config.ServerMap)),
	}

	res, err := s3svc.GetObject(s3GetInfo)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println(res)
	body, bodyErr := ioutil.ReadAll(res.Body)
	if bodyErr != nil {
		fmt.Println(err)
	}
	fmt.Println("...getServerMapFileFromS3")
	return body
}

func getServerMapFile(s3Config *S3Config) ServerMap {

	var body []byte

	if s3Config != nil {
		body = getServerMapFileFromS3(s3Config)
	}
	//TODO Add other potential options for where the file might live...

	var sm ServerMap
	jErr := json.Unmarshal(body, &sm)
	if jErr != nil {
		fmt.Println("ERROR IN UNMARSHAL")
		fmt.Println(jErr)
		//panic(jErr)
		os.Exit(1)
	}
	fmt.Println("sm: ", sm)

	return sm
}

func ProcessServerMap(sm ServerMap) {
	var wg sync.WaitGroup
	for i, dom := range sm {
		log.Println("psm: ", i, dom)

		req, err := http.NewRequest("GET", "", nil)
		if err != nil {
			log.Print(err)
			os.Exit(1)
		}

		req.URL.Scheme = "https"
		req.URL.Host = dom.DomainName
		req.URL.Path = dom.URLPath

		qs := req.URL.Query()
		for k, v := range dom.Qs {
			qs.Add(k, v)
		}
		req.URL.RawQuery = qs.Encode()
		// fmt.Println(i, "Adding...")
		wg.Add(1)
		go func(url string, r ReqRtn) {
			defer wg.Done()
			var res = GrabURLData(url)
			if res == nil {
				log.Println("Got an invalid result back...go to error matching functions!")
				return
			}
			//log.Println(res.StatusCode)
			if res.StatusCode != r.Code {
				log.Println("ERROR - Mismatched status code! ", url)
			}
			body, bodyErr := ioutil.ReadAll(res.Body)
			if bodyErr != nil {
				log.Println("ERROR - BodyErr: ", err)
			}

			if r.Val != "" { // if it's empty we just don't care what's in it...
				bodyStr := string(body)
				if bodyStr != r.Val {
					log.Println(bodyStr)
					log.Println("ERROR - Wrong text value! ", url, " r.Val ", r.Val, " bodyStr ", bodyStr)
				}
			}
			defer res.Body.Close()
		}(req.URL.String(), dom.Rtn)
	}
	// fmt.Println("waiting...")
	wg.Wait()
}

func init() {}

func lambdaHandler(ctx context.Context, cwe CloudWatchEvent) {
	lc, _ := lambdacontext.FromContext(ctx)
	log.Println(lc.AwsRequestID)
	log.Println("lc", lc)
	log.Println("cwe", cwe)
	

	var startTime = time.Now()
	var s3config S3Config
	jErr := json.Unmarshal(cwe.Detail, &s3config)
	if jErr != nil {
		log.Println("ERROR IN s3config UNMARSHAL")
		log.Println(jErr)
		os.Exit(1)
	}
	log.Print("s3config: ", s3config)
	timing("s3config", startTime)
	
	startTime = time.Now() // reset...
	sm := getServerMapFile(&s3config)
	log.Print("ServerMap", sm)
	timing("ServerMap", startTime)

	startTime = time.Now() // reset...
	ProcessServerMap(sm)
	log.Println("ServerMap processed!")
	timing("ServerMap processing", startTime)
	return
}

func main() {
	log.Print("Starting lambda handler...")
	lambda.Start(lambdaHandler)
}
