package lib

import (
	"encoding/json"
	_ "expvar"
	"fmt"
	"io/ioutil"
	"net/http"
	_ "net/http/pprof"
	"os"
	"sync"
	"log"

	// "github.com/aws/aws-lambda-go/lambda"
	// "github.com/aws/aws-lambda-go/lambdacontext"
)

func GetServerMapFile(s3Config *S3Config) ServerMap {

	var body []byte

	if s3Config != nil {
		body = GetServerMapFileFromS3(s3Config)
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