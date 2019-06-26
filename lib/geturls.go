package lib

import (
	"crypto/tls"
	_ "expvar"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"log"
)

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