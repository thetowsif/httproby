package main

import (
	"bufio"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
)

func main() {
	listPath := flag.String("list", "", "file containing list of domains")
	workers := flag.Int("workers", 20, "number of worker threads")
	flag.Parse()

	if *listPath == "" {
		fmt.Println("Please provide a file containing list of domains with the -list flag")
		return
	}

	// Open the domain list file
	listFile, err := os.Open(*listPath)
	if err != nil {
		fmt.Printf("Error opening %s: %s\n", *listPath, err)
		return
	}
	defer listFile.Close()

	domains := make(chan string, 100)

	// Start worker threads
	var wg sync.WaitGroup
	for i := 0; i < *workers; i++ {
		wg.Add(1)
		go worker(domains, &wg)
	}

	// Read the domain list file and send each domain to the worker threads
	scanner := bufio.NewScanner(listFile)
	for scanner.Scan() {
		domains <- scanner.Text()
	}
	close(domains)

	wg.Wait()
}

func worker(domains <-chan string, wg *sync.WaitGroup) {
	defer wg.Done()

	for domain := range domains {
		for _, protocol := range []string{"https://", "http://"} {
			url := protocol + domain
			redirectedDomain := probeURL(url, 10)

			if redirectedDomain != "" {
				fmt.Println("https://" + redirectedDomain)
				break
			}
		}
	}
}

func probeURL(urlStr string, maxRedirects int) string {
	redirectedDomain := ""
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return fmt.Errorf("stopped after %d redirects", maxRedirects)
			}
			return nil
		},
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(urlStr)
	if err != nil {
		return redirectedDomain
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		// We have reached the final URL, return the domain name
		redirectedDomain = getDomain(resp.Request.URL.String())
	} else if resp.StatusCode >= http.StatusMultipleChoices && resp.StatusCode <= http.StatusPermanentRedirect {
		// The URL redirected, try to probe the new location
		location, err := resp.Location()
		if err != nil {
			return redirectedDomain
		}
		redirectedDomain = probeURL(location.String(), maxRedirects-1)
	}

	return redirectedDomain
}

// func getDomain(urlStr string) string {
// 	u, err := url.Parse(urlStr)
// 	if err != nil {
// 		return ""
// 	}

// 	domainParts := strings.Split(u.Hostname(), ".")
// 	if len(domainParts) >= 2 {
// 		return domainParts[len(domainParts)-2] + "." + domainParts[len(domainParts)-1]
// 	}
// 	return ""
// }

func getDomain(urlStr string) string {
    u, err := url.Parse(urlStr)
    if err != nil {
        return ""
    }

    return u.Hostname()
}
