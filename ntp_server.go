package main

import (
	"encoding/hex"
	"log"
	"net"
	"time"
)

// NTP服务器监听地址和端口
const listenAddress = "0.0.0.0:123"

func startNTPServer() {
	addr, err := net.ResolveUDPAddr("udp", listenAddress)
	if err != nil {
		log.Fatalf("Failed to resolve UDP address: %v", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatalf("Failed to listen on UDP address: %v", err)
	}
	defer conn.Close()
	log.Println("NTP server listening on", listenAddress)

	buffer := make([]byte, 48)
	for {
		n, clientAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Printf("Error reading from UDP connection: %v", err)
			continue
		}

		// 处理客户端请求并返回时间
		go handleClient(conn, clientAddr, buffer[:n])
	}
}

// 处理客户端请求
func handleClient(conn *net.UDPConn, clientAddr *net.UDPAddr, request []byte) {
	log.Printf("Recv: %s", hex.EncodeToString(request))
	// 简单回应客户端当前时间
	currentTime := getCurrentTime()
	response := make([]byte, 48)
	// 填充NTP响应包，这里简化了，只设置时间戳相关字段
	secs := uint32(currentTime.Unix() + 2208988800) // NTP时间从1900年开始计算
	nanos := uint32(currentTime.UnixNano() % 1e9)
	response[0] = 0x1c // LI, Version, Mode
	response[43] = byte(secs)
	response[42] = byte(secs >> 8)
	response[41] = byte(secs >> 16)
	response[40] = byte(secs >> 24)
	response[47] = byte(nanos)
	response[46] = byte(nanos >> 8)
	response[45] = byte(nanos >> 16)
	response[44] = byte(nanos >> 24)

	_, err := conn.WriteToUDP(response, clientAddr)
	if err != nil {
		log.Printf("Failed to send response to %v: %v", clientAddr, err)
	}
}

// 获取当前时间，如果上次同步超过一小时，尝试重新同步
func getCurrentTime() time.Time {
	if time.Since(lastSyncTime) > time.Hour {
		if err := syncTimeFromNTP(); err != nil {
			log.Printf("Failed to sync time from NTP server, using local time: %v", err)
		}
	}
	return currentTime
}
