package main

import (
	"context"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/alexflint/go-arg"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/stealth"
)

type Args struct {
	URL            string `arg:"positional,required" help:"The starting URL for the crawler."`
	Verbose        bool   `arg:"-v,--verbose" help:"Enable verbose logging."`
	DisableCookies bool   `arg:"--disable-cookies" help:"Disable cookies with requests."`
	LogFile        string `arg:"--log" help:"Log output to the specified file."`
	OutputFilename string `arg:"-o,--output" help:"Output file for saving scraped emails." default:"emails.txt"`
	SkipValidation bool   `arg:"--skip-validation" help:"Skip email validation."`
	UserAgent      string `arg:"--user-agent" help:"Custom User-Agent for requests."`
	MaxDepth       int    `arg:"--max-depth" help:"Maximum crawling depth." default:"3"`
	DomainMode     int    `arg:"--domain-mode" help:" 1 to stay within current site, 2 to explore subdirectories, 3 for unrestricted" default:"1"`
}

func main() {
	var args Args
	arg.MustParse(&args)

	if args.Verbose {
		log.Printf("Starting web crawler with URL: %s\n", args.URL)
	}
	if args.LogFile != "" {
		logFile, err := os.OpenFile(args.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalf("Failed to open log file: %v", err)
		}
		defer logFile.Close()
		log.SetOutput(logFile)
	}

	outputFile, err := os.Create(args.OutputFilename)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer outputFile.Close()

	launcher := launcher.New().Headless(true)
	if args.DisableCookies {
		launcher.Set("--disable-cookies")
	}
	if args.UserAgent != "" {
		launcher.Set("--user-agent=", args.UserAgent)
	}
	browser := rod.New().ControlURL(launcher.MustLaunch()).MustConnect()
	defer browser.MustClose()
	browser.HijackRequests()
	page := stealth.MustPage(browser)

	var emailsOutput []string
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	crawl(ctx, page, args, &emailsOutput, args.MaxDepth, 1)

	for _, email := range emailsOutput {
		if args.SkipValidation || ValidateEmail(email) {
			if args.Verbose {
				log.Printf("Valid email found: %s\n", email)
			}
			_, _ = outputFile.WriteString(email + "\n")
		}
	}

	log.Printf("Saved %d emails to %s\n", len(emailsOutput), args.OutputFilename)
}

func crawl(ctx context.Context, page *rod.Page, args Args, emailsOutput *[]string, maxDepth, currentDepth int) {
	if currentDepth > maxDepth {
		return
	}

	var e proto.NetworkResponseReceived

	wait := page.WaitEvent(&e)
	err := page.Navigate(args.URL)
	if err != nil {
		log.Printf("Failed to open page for URL %s: %v\n", args.URL, err)
		return
	}
	wait()

	html, err := page.HTML()
	if err != nil || strings.TrimSpace(html) == "" {
		log.Printf("Failed to retrieve or empty page HTML for URL %s: %v\n", args.URL, err)
		_ = page.Close()
		return
	}

	emails, _ := GetEmails(html, args.URL)
	for _, email := range emails {
		*emailsOutput = append(*emailsOutput, email)
	}

	links := ExtractLinks(html, args.URL, args.DomainMode)
	for link := range links {
		select {
		case <-ctx.Done():
			return
		default:
			crawl(ctx, page, Args{URL: link}, emailsOutput, maxDepth, currentDepth+1)
		}
	}
}

func ExtractLinks(html string, baseURL string, domainMode int) map[string]struct{} {
	linkSet := make(map[string]struct{})

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		log.Printf("Failed to parse HTML: %v\n", err)
		return linkSet
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		log.Printf("Invalid base URL: %v\n", err)
		return linkSet
	}

	doc.Find("a, iframe, frame").Each(func(i int, s *goquery.Selection) {
		var link string
		if href, exists := s.Attr("href"); exists {
			link = href
		} else if src, exists := s.Attr("src"); exists {
			link = src
		} else {
			return
		}

		var absoluteURL string
		parsedURL, err := url.Parse(link)
		if err != nil {
			return
		}
		if parsedURL.IsAbs() {
			absoluteURL = parsedURL.String()
		} else {
			absoluteURL = base.ResolveReference(parsedURL).String()
		}

		if absoluteURL == "" {
			return
		}

		switch domainMode {
		case 1:
			if !strings.HasPrefix(absoluteURL, base.Scheme+"://"+base.Host) {
				return
			}
		case 2:
			if !strings.HasPrefix(absoluteURL, base.Scheme+"://"+base.Host+base.Path) {
				return
			}
		case 3:
			// No restrictions
		}
		linkSet[absoluteURL] = struct{}{}
	})

	return linkSet
}
