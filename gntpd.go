package main

import (
	"fmt"
	"log"
	"time"
)

var currentTime time.Time
var lastSyncTime time.Time

func main() {
	fmt.Println("gNTPd started ...")
	if err := syncTimeFromNTP(); err != nil {
		log.Printf("Failed to sync time from NTP server,using local time :%v", err)
	}
	startNTPServer()
}
