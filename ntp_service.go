package main

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math"
	"net"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/beevik/ntp"
)

var (
	gWG             sync.WaitGroup
	gStopChan       = make(chan struct{})
	gMutexLocalTime sync.Mutex
)

func syncTimeFromNTP() (time.Time, error) {
	ret := time.Now()
	var retErr error
	startTime := time.Now()
	if response, err := ntp.Time(gNTPServer); err != nil {
		retErr = fmt.Errorf("failed to get time from ntp server: %v", err)
	} else {
		log.Printf("Time synchronized from NTP server: %v\t Cost:%v ms", response, time.Since(startTime).Milliseconds())
		ret = response
	}
	return ret, retErr
}

func threadUDPService() {
	defer gWG.Done()
	listenAddress := fmt.Sprintf("0.0.0.0:%d", gServicePort)
	if addr, err := net.ResolveUDPAddr("udp", listenAddress); err != nil {
		log.Printf("Failed to resolve UDP address: %v", err)
	} else if conn, err := net.ListenUDP("udp", addr); err != nil {
		log.Printf("Failed to listen on UDP address: %v", err)
	} else {
		defer func(conn *net.UDPConn) {
			err := conn.Close()
			if err != nil {
				log.Println(err)
			}
		}(conn)
		log.Println("NTP server listening on", listenAddress)

		buffer := make([]byte, 48)
		for {
			select {
			case <-gStopChan:
				log.Println("NTP server(udp) stopping...")
				return
			default:
				if errd := conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond)); errd != nil {
					log.Printf("Failed to set read deadline: %v", errd)
				} else {
					n, clientAddr, err := conn.ReadFromUDP(buffer)
					var neterr net.Error
					if errors.As(err, &neterr) && neterr.Timeout() {
						// 读取超时，继续检查 gStopChan
						continue
					}

					// 处理客户端请求并返回时间
					go handleClient(conn, clientAddr, buffer[:n])
				}
			}
		}
	}
}

func threadSyncTime() {
	defer gWG.Done()
	ticker := time.NewTicker(time.Duration(gInterval) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-gStopChan:
			log.Println("NTP server stopping...")
			return
		case <-ticker.C:
			clientSyncTime()
		}
	}
}

func startNTPServer() {
	clientSyncTime()
	gWG.Add(1)
	go threadSyncTime()
	if gServerMode {
		gWG.Add(1)
		go threadUDPService()
	}
}

func clientSyncTime() {
	if t, err := syncTimeFromNTP(); err != nil {
		log.Printf("Failed to sync time from NTP server: %v", err)
	} else {
		log.Printf("Local time: %v\nNTP time: %v\nSince:%f sec", time.Now(), t, time.Since(t).Seconds())
		if math.Abs(time.Since(t).Seconds()) > 1 { // ntp时间与本地时间相差大于1s
			log.Printf("Time difference is greater than 1 second, resetting local time.\nNTP time: %v\nLocal time: %v", t, time.Now())
			// 设置本地时间
			go setLocalTime(t)
		}
	}
}

func stopNTPServer() {
	close(gStopChan)
	gWG.Wait()
	log.Println("NTP server stopped")
}

// 处理客户端请求
func handleClient(conn *net.UDPConn, clientAddr *net.UDPAddr, request []byte) {
	if len(request) < 48 {
		log.Printf("Invalid request from %v: insufficient data", clientAddr)
		return
	}

	// 创建NTP响应
	response := make([]byte, 48)

	// 设置Leap Indicator, Version Number, Mode
	response[0] = 0x24 // LI = 0, VN = 4, Mode = 4 (server)

	// 设置Stratum
	response[1] = 1 // Stratum = 1 (primary server)

	// 设置Poll Interval
	response[2] = 4 // Poll Interval = 4 (16 seconds)

	// 设置Precision
	response[3] = 0xfa // Precision = -6 (15.625ms)

	// 设置Root Delay
	binary.BigEndian.PutUint32(response[4:], uint32(0)) // Root Delay = 0

	// 设置Root Dispersion
	binary.BigEndian.PutUint32(response[8:], uint32(0)) // Root Dispersion = 0

	// 设置Reference Identifier
	binary.BigEndian.PutUint32(response[12:], uint32(0)) // Reference Identifier = 0

	// 设置Reference Timestamp
	nowTime := time.Now()
	log.Printf("Recv from %v:\n%s\nNow time: %v", clientAddr, hex.EncodeToString(request), nowTime)
	secs := uint32(nowTime.Unix() + 2208988800) // convert from 1970 epoch to 1900 epoch
	frac := uint32((nowTime.UnixNano() % 1e9) << 32 / 1e9)
	binary.BigEndian.PutUint32(response[16:], secs)
	binary.BigEndian.PutUint32(response[20:], frac)

	// 设置Originate Timestamp (copy from request)
	copy(response[24:32], request[40:48])

	// 设置Receive Timestamp
	binary.BigEndian.PutUint32(response[32:], secs)
	binary.BigEndian.PutUint32(response[36:], frac)

	// 设置Transmit Timestamp
	binary.BigEndian.PutUint32(response[40:], secs)
	binary.BigEndian.PutUint32(response[44:], frac)

	// 发送响应
	if _, err := conn.WriteToUDP(response, clientAddr); err != nil {
		log.Printf("Failed to send response to %v: %v", clientAddr, err)
		return
	}
}

func setLocalTime(t time.Time) {
	gMutexLocalTime.Lock()
	defer gMutexLocalTime.Unlock()
	if erru := UpdateSystemDate(fmt.Sprintf("%d-%d-%d %d:%d:%d", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())); erru != nil {
		log.Printf("Set local time to %v failed: %v", t, erru)
	} else {
		log.Printf("Set local time to %v successfully", t)
	}
}

// UpdateSystemDate SetSystemDate set system date,eg: 2019-01-01 00:00:00
func UpdateSystemDate(dateTime string) error {
	var err error
	system := runtime.GOOS
	switch system {
	case "windows":
		date := strings.Split(dateTime, " ")
		if len(date) != 2 {
			err = fmt.Errorf("invalid date time: %s", dateTime)
		} else {
			cmd := exec.Command("PowerShell.exe", fmt.Sprintf("date %s", date[0]))
			if output, erro := cmd.Output(); nil != erro {
				err = fmt.Errorf("set date error: %s", string(output))
			} else {
				cmd = exec.Command("PowerShell.exe", fmt.Sprintf("time %s", date[1]))
				if output, erro := cmd.Output(); nil != erro {
					err = fmt.Errorf("set time error: %s", string(output))
				}
			}
		}

	case "linux":
		cmd := exec.Command("date", "-s", dateTime)
		if output, erro := cmd.Output(); nil != erro {
			err = fmt.Errorf("set date error: %s", string(output))
		}
	default:
		err = fmt.Errorf("unsupport system: %s", system)
	}
	return err
}
