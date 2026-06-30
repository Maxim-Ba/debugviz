package main

import (
	"log"

	"github.com/robfig/cron/v3"
)

func main() {
	c := cron.New()
	_, _ = c.AddFunc("@every 1h", syncReports)
	c.Start()
	log.Println("cron worker started")
}

func syncReports() {
	log.Println("sync reports")
}
