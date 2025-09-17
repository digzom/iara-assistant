package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	DOMBaseURL     = "https://dom.mossoro.rn.gov.br/dom"
	WebhookURL     = "https://iara.digzom.dev/webhook/f97912c2-a20c-45c5-9642-4e51d33bd7d9/selection-process"
	RequestTimeout = 30 * time.Second
)

var targetKeywords = []string{
	"convocação",
	"processo seletivo",
	"processo seletivo simplificado",
	"Edital nº 01/2025 da Secretaria Municipal de Educação",
	"Secretaria Municipal de Educação",
}

type DOMCrawler struct {
	httpClient *http.Client
}

type WebhookPayload struct {
	URL    string `json:"url"`
	RawDoc string `json:"rawDoc"`
}

func NewDOMCrawler() *DOMCrawler {
	return &DOMCrawler{
		httpClient: &http.Client{
			Timeout: RequestTimeout,
		},
	}
}

func (c *DOMCrawler) CrawlDOM() error {
	log.Println("Starting DOM crawl...")

	// Step 1: Get the main DOM page
	resp, err := c.httpClient.Get(DOMBaseURL)
	if err != nil {
		return fmt.Errorf("failed to fetch DOM main page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("DOM main page returned status: %d", resp.StatusCode)
	}

	// Step 2: Parse HTML and find the publication link
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to parse HTML: %w", err)
	}

	number, err := c.extractLastPublicationNumber(doc)
	if err != nil {
		return fmt.Errorf("can't find that fucking number, sorry bro")
	}

	lastSavedNumber, err := c.getLastSavedNumber()
	if err != nil {
		return fmt.Errorf("can't get the last saved number :(")
	}

	intNumber, err := strconv.Atoi(number)
	if err != nil {
		return fmt.Errorf("gone wrong")
	}

	intLastSavedNumber, err := strconv.Atoi(lastSavedNumber)
	if err != nil {
		return fmt.Errorf("gone wrong 2")
	}

	if intNumber <= intLastSavedNumber {
		return fmt.Errorf("it's not an error, but no need to send the webhook now :)")
	}

	publicationLink, err := c.extractPublicationLink(doc)
	if err != nil {
		return fmt.Errorf("failed to extract publication link: %w", err)
	}

	log.Printf("Found publication link: %s", publicationLink)

	// Step 3: Visit the publication page and check for keywords
	fullPublicationURL := "https://dom.mossoro.rn.gov.br" + publicationLink

	hasKeywords, err := c.checkForKeywords(fullPublicationURL)
	if err != nil {
		return fmt.Errorf("failed to check keywords: %w", err)
	}

	// Step 4: Send webhook notification if keywords found
	if hasKeywords {
		log.Printf("Keywords found! Sending webhook notification for: %s", fullPublicationURL)
		if err := c.sendWebhook(fullPublicationURL, doc.Text()); err != nil {
			return fmt.Errorf("failed to send webhook: %w", err)
		}

		err = os.WriteFile("last_dom", []byte(number), 0644)
		if err != nil {
			return fmt.Errorf("cant write shit")
		}

		log.Println("Webhook sent successfully")
	} else {
		log.Println("No target keywords found in this publication")
	}

	return nil
}

func (c *DOMCrawler) getLastSavedNumber() (string, error) {
	_, err := os.Stat("last_dom")
	if err != nil {
		os.WriteFile("last_dom", []byte("0"), 0644)
	}

	content, err := os.ReadFile("last_dom")
	if err != nil {
		return "", fmt.Errorf("sorry bro, cant open this shit")
	}

	return string(content), nil
}

func (c *DOMCrawler) extractLastPublicationNumber(doc *goquery.Document) (string, error) {
	var publicationNumber string

	doc.Find(".jom-title").Each(func(i int, h3 *goquery.Selection) {
		title := h3.Find("a").First().Text()
		publicationNumber = strings.TrimPrefix(title, "DOM Nº ")
	})

	if publicationNumber == "" {
		return "", fmt.Errorf("coud not find publication number")
	}

	return publicationNumber, nil
}

func (c *DOMCrawler) extractPublicationLink(doc *goquery.Document) (string, error) {
	// Look for the div with class "last-jom-actions" under section with id "ultima-edicao"
	var publicationLink string

	doc.Find("#ultima-edicao").Each(func(i int, section *goquery.Selection) {
		section.Find(".last-jom-actions").Each(func(j int, div *goquery.Selection) {
			// Get the first <a> tag (Leitura Online)
			firstLink := div.Find("a").First()
			if href, exists := firstLink.Attr("href"); exists {
				publicationLink = href
			}
		})
	})

	if publicationLink == "" {
		return "", fmt.Errorf("could not find publication link in last-jom-actions div")
	}

	return publicationLink, nil
}

func (c *DOMCrawler) checkForKeywords(url string) (bool, error) {
	log.Printf("Checking for keywords in: %s", url)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return false, fmt.Errorf("failed to fetch publication page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("publication page returned status: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to parse publication HTML: %w", err)
	}

	// Get all text content from the page
	pageText := strings.ToLower(doc.Text())

	// Check for each target keyword
	for _, keyword := range targetKeywords {
		normalizedKeyword := strings.ToLower(keyword)
		if strings.Contains(pageText, normalizedKeyword) {
			log.Printf("Found keyword: %s", keyword)
			return true, nil
		}
	}

	return false, nil
}

func (c *DOMCrawler) sendWebhook(url string, doc string) error {
	payload := WebhookPayload{
		URL:    url,
		RawDoc: doc,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	resp, err := c.httpClient.Post(WebhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send webhook request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned non-success status: %d", resp.StatusCode)
	}

	return nil
}
