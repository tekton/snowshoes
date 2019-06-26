package lib

import (
	"time"
	"log"
)

// similar to a util file, but named different

func Timing(s string, startTime time.Time) {
    endTime := time.Now()
    log.Println(s, "took", endTime.Sub(startTime))
}