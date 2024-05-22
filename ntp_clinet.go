package main

import (
	"log"
	"time"

	"github.com/beevik/ntp"
)

// 配置上游NTP服务器地址
const ntpServer = "pool.ntp.org"

func syncTimeFromNTP() error {
	if response, err := ntp.Time(ntpServer); err != nil {
		return err
	} else {
		currentTime = response
		lastSyncTime = time.Now()
		log.Println("Time synchronized from NTP server:", currentTime)
	}
	return nil
}
