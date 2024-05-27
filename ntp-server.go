package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

	ks "github.com/kardianos/service"
)

type JsonNTPConfig struct {
	NTPServer     string `json:"ntp_server"`
	NTPServerPort int64  `json:"ntp_server_port"`
	ServerMode    bool   `json:"server_mode"`
	ServicePort   int64  `json:"service_port"`
	Interval      int64  `json:"interval"`
	Version       string `json:"version"`
}

const configFile = "config.json"

var gNTPServer = "pool.ntp.org:123"
var gServerMode = false
var gServicePort int64 = 123
var gInterval int64 = 60 // seconds

// loadConfig loads the configuration from the config file,config.json
func loadConfig() error {
	var jCfg JsonNTPConfig
	var retErr error
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		retErr = fmt.Errorf("config file %s not found", configFile)
	} else if file, erro := os.Open(configFile); erro != nil {
		retErr = fmt.Errorf("error opening config file: %s", err)
	} else if content, errr := io.ReadAll(file); errr != nil {
		retErr = fmt.Errorf("error reading config file: %s", errr)
	} else if erru := json.Unmarshal(content, &jCfg); erru != nil {
		retErr = fmt.Errorf("error parsing config file: %s", erru)
	} else {
		gNTPServer = fmt.Sprintf("%s:%d", jCfg.NTPServer, jCfg.NTPServerPort)
		gServerMode = jCfg.ServerMode
		if jCfg.ServicePort > 0 {
			gServicePort = jCfg.ServicePort
		}
		if jCfg.Interval > 0 {
			gInterval = jCfg.Interval
		}
		fmt.Printf("NTP server configuration loaded successfully.\n"+
			"NTP Server: %s\nServer Mode: %v\nService Port: %d\nInterval: %d\nVersion: %s\n",
			gNTPServer, gServerMode, gServicePort, gInterval, jCfg.Version)
	}
	return retErr
}

type program struct{}

func (p *program) Start(_ ks.Service) error {
	go p.run()
	return nil
}
func (p *program) run() {
	localStart()
}

func (p *program) Stop(_ ks.Service) error {
	stopNTPServer()
	return nil
}

func localStart() {
	if errl := loadConfig(); errl != nil {
		log.Println(errl)
	}
	go startNTPServer()
}

func main() {
	fmt.Println("gNTPd started ...")
	if len(os.Args) > 1 && os.Args[1] == "normal" {
		log.Println("Run mode: normal")
		localStart()
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh
		log.Println("Stopping...")
		stopNTPServer()
		log.Println("End...")
	} else {
		svcConfig := &ks.Config{
			Name:        "gNTPd",
			DisplayName: "gNTPd",
			Description: "gNTPd is a simple NTP server",
		}
		prg := &program{}
		if s, err := ks.New(prg, svcConfig); err != nil {
			log.Println(err)
		} else {
			if len(os.Args) > 1 {
				switch os.Args[1] {
				case "install":
					if err := s.Install(); err != nil {
						log.Println(err)
					} else {
						log.Println("service install success!")
					}
					if err := s.Start(); err != nil {
						log.Println(err)
					} else {
						log.Println("service start success!")
					}
				case "start":
					if err := s.Start(); err != nil {
						log.Println(err)
					} else {
						log.Println("service start success")
					}
				case "stop":
					if err := s.Stop(); err != nil {
						log.Println(err)
					} else {
						log.Println("service stop success!")
					}
				case "restart":
					if err := s.Stop(); err != nil {
						log.Println(err)
					} else {
						log.Println("service stop success!")
					}
					if err := s.Start(); err != nil {
						log.Println(err)
					} else {
						log.Println("service start success!")
					}
				case "remove", "uninstall":
					if err := s.Stop(); err != nil {
						log.Println(err)
					} else {
						log.Println("service stop success!")
					}
					if err := s.Uninstall(); err != nil {
						log.Println(err)
					} else {
						log.Println("service uninstall success!")
					}
				default:
					log.Println("unknown command")
				}
				return
			}
			if err = s.Run(); err != nil {
				log.Println(err)
			}
		}
	}
}
