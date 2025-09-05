package services

import (
	"log"
	"time"

	"github.com/robfig/cron/v3"
)

type CronService struct {
	cron    *cron.Cron
	crawler *DOMCrawler
}

func NewCronService() *CronService {
	// Create cron with timezone support
	location, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		log.Printf("Warning: Could not load timezone, using UTC: %v", err)
		location = time.UTC
	}

	c := cron.New(cron.WithLocation(location))
	crawler := NewDOMCrawler()

	return &CronService{
		cron:    c,
		crawler: crawler,
	}
}

func (cs *CronService) Start() error {
	// Schedule the crawler to run every 2 hours from 08:00 to 23:59
	// Cron expression: "0 8-23/2 * * *" means every 2 hours from 8 to 23 (8, 10, 12, 14, 16, 18, 20, 22)
	_, err := cs.cron.AddFunc("0 8-23/2 * * *", func() {
		log.Println("DOM crawler cron job triggered")
		if err := cs.crawler.CrawlDOM(); err != nil {
			log.Printf("DOM crawler error: %v", err)
		}
	})
	if err != nil {
		return err
	}

	log.Println("DOM crawler cron service started - running every 2 hours from 8AM to 11PM")
	cs.cron.Start()
	return nil
}

func (cs *CronService) Stop() {
	log.Println("Stopping DOM crawler cron service")
	cs.cron.Stop()
}

// Manual trigger for testing purposes
func (cs *CronService) TriggerCrawler() error {
	log.Println("Manual DOM crawler trigger")
	return cs.crawler.CrawlDOM()
}