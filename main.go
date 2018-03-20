package main

import (
	"crypto/tls"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"io/ioutil"
	"net/http"
	"os"
	"encoding/json"
	"sync"
	"github.com/spf13/viper"
	log "github.com/sirupsen/logrus"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
	_ "expvar"
	_ "net/http/pprof"
	"context"
	"github.com/aws/aws-lambda-go/lambdacontext"
)

// application wide settings
var SETTINGS *viper.Viper
var LOGGER *log.Logger

type ReqRtn struct {
	Code int `json:"code"`
	Type string `json:"type"`
	Val string `json:"val"`
}

type ServerMap []struct {
	DomainTypeId int `json:"domain_type_id"`
	Val string `json:"val"`
	Rtn ReqRtn `json:"rtn"`
	ClientID int `json:"client_id"`
	URLPath string `json:"url_path"`
	Qs map[string]string `json:"qs,omitempty"`
	Grouping string `json:"grouping"`
	DomainName string `json:"domain_name"`
}

type S3Config struct {
	Bucket string
	Prefix string
	ServerMap string
	Region string
}

func GrabURLData(url string) *http.Response {
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
		LOGGER.Error("GET ERROR", err)
		return nil
	}
	//defer res.Body.Close()
	return res
}

// TODO: Validate response string
// TODO: ADD "Actions" support for success and error
// TODO: Base integrations for PagerDuty and Slack (mostly for actions)
// TODO: Logging
// TODO: Lambda run support- default config?
// TODO: Validate SSL


func getServerMapFileFromS3(s3Config *S3Config) []byte {
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
	fmt.Println(sm)

	return sm
}

func ProcessServerMap(sm ServerMap) {
	var wg sync.WaitGroup
	for i, dom := range sm {
		fmt.Println(i, dom)

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
		wg.Add(1)
		go func(url string, r ReqRtn) {
			defer wg.Done()
			var res = GrabURLData(url)
			if res == nil {
				LOGGER.Error("Got an invalid result back...go to error matching functions!")
				return
			}
			//log.Println(res.StatusCode)
			if res.StatusCode != r.Code {
				LOGGER.Error("ERROR - Mismatched status code!", url)
			}
			body, bodyErr := ioutil.ReadAll(res.Body)
			if bodyErr != nil {
				LOGGER.Error("ERROR - BodyErr: ", err)
			}
			bodyStr := string(body)
			LOGGER.Debug(bodyStr)
			LOGGER.Error("ERROR - Wrong text value!", url, "r.Val", r.Val, "bodyStr", bodyStr)
			defer res.Body.Close()
		}(req.URL.String(), dom.Rtn)
	}
	wg.Wait()
}

func init() {
	SETTINGS = viper.New()
	SETTINGS.Set("verbose", true)
	SETTINGS.SetConfigName("config")		// should be a json file
	SETTINGS.AddConfigPath(".")             // when all else, look local
	viper_err := SETTINGS.ReadInConfig()    // Find and read the config file
	if viper_err != nil {                   // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error config file: %s \n", viper_err))
	} else {
		fmt.Println(SETTINGS.AllKeys())
	}
	// Setup logger for future logging...
	LOGGER = log.New()
	//LOGGER.Formatter = &utilities.JSONFormatter{}
	LOGGER.Formatter = &log.JSONFormatter{}
	if SETTINGS.IsSet("logs.level") {
		if SETTINGS.GetString("logs.level") == "debug" {
			LOGGER.SetLevel(log.DebugLevel)
		} else {
			LOGGER.SetLevel(log.InfoLevel)
		}
	}
	LOGGER.Out = &lumberjack.Logger{
		Filename: SETTINGS.GetString("logs.default"),
		MaxSize:  5,
		MaxAge:   1,
	}
}

func lambdaHandler(ctx context.Context) {
	lc, _ := lambdacontext.FromContext(ctx)
	fmt.Print(lc.AwsRequestID)
	fmt.Print(lc)
}

func main() {
	s3Config := &S3Config{
		Bucket: SETTINGS.GetString("Bucket"),
		Prefix: SETTINGS.GetString("Prefix"),
		ServerMap: SETTINGS.GetString("ServerMap"),
		Region: SETTINGS.GetString("Region"),
	}
	sm := getServerMapFile(s3Config)
	ProcessServerMap(sm)
	lambda.Start(lambdaHandler)
}
