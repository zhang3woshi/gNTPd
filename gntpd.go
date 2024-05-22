package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	ks "github.com/kardianos/service"
)

var currentTime time.Time
var lastSyncTime time.Time

type program struct{}

func (p *program) Start(s ks.Service) error {
	go p.run()
	return nil
}
func (p *program) run() {
	go localStart()
}

func (p *program) Stop(s ks.Service) error {
	stopNTPServer()
	return nil
}

func localStart() {
	if err := syncTimeFromNTP(); err != nil {
		log.Printf("Failed to sync time from NTP server,using local time :%v", err)
	}
	startNTPServer()
}

func main() {
	fmt.Println("gNTPd started ...")
	if len(os.Args) > 1 && os.Args[1] == "normal" {
		log.Println("Run mode: normal")
		go localStart()
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
