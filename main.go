package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/urfave/cli/v3"
)

func checkVerification(link string, context playwright.BrowserContext) (bool, error) {
	newPage, err := context.NewPage()
	if err != nil {
		log.Printf("could not create new page: %v", err)
		return false, err
	}
	defer newPage.Close()

	_, err = newPage.Goto(link, playwright.PageGotoOptions{
		Timeout:   playwright.Float(5000),
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	})
	if err != nil {
		newPage.Close()
		return false, err
	}

	rateLimitLocator := newPage.Locator("text=You are rate limited")
	isRateLimited, _ := rateLimitLocator.IsVisible(playwright.LocatorIsVisibleOptions{
		Timeout: playwright.Float(150),
	})
	if isRateLimited {
		println("Rate limit reached, waiting 10s...")
		time.Sleep(10 * time.Second)
		newPage.Reload()
	}

	// Quick check
	verifiedLocator := newPage.Locator("h4:has-text('Dostępna Weryfikacja Przedmiotów')")
	isVerified, err := verifiedLocator.IsVisible(playwright.LocatorIsVisibleOptions{
		Timeout: playwright.Float(100),
	})

	// Second check with longer timeout
	if err != nil || !isVerified {
		isVerified, _ = verifiedLocator.IsVisible(playwright.LocatorIsVisibleOptions{
			Timeout: playwright.Float(300),
		})
	}

	return isVerified, nil
}

func getLinks(url string, logs bool) []string {

	pw, err := playwright.Run()
	if err != nil {
		log.Fatalf("could not start Playwright: %v", err)
	}
	defer pw.Stop()

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
	})
	if err != nil {
		log.Fatalf("could not launch browser: %v", err)
	}
	defer browser.Close()

	context, err := browser.NewContext(playwright.BrowserNewContextOptions{
		Viewport: &playwright.Size{
			Width:  1280,
			Height: 720,
		},
	})
	if err != nil {
		log.Fatalf("could not create context: %v", err)
	}
	defer context.Close()

	page, err := browser.NewPage()
	if err != nil {
		log.Fatalf("could not create page: %v", err)
	}

	_, err = page.Goto(url)
	if err != nil {
		log.Fatalf("could not go to page: %v", err)
	}

	locator := page.Locator("a[href^='https://www.vinted.pl/items/']")

	err = locator.Nth(0).WaitFor(playwright.LocatorWaitForOptions{Timeout: playwright.Float(5000)})
	if err != nil {
		log.Fatalf("could not find the selector: %v", err)
	}

	var links []string

	count, err := locator.Count()
	if err != nil {
		log.Fatalf("could not get count of elements: %v", err)
	}

	fmt.Println("Main website succesfully scraped!\n")

	for i := 0; i < count; i++ {
		link, err := locator.Nth(i).GetAttribute("href")
		if err != nil {
			log.Printf("could not get attribute for item %d: %v", i, err)
			continue
		}

		isVerified, err := checkVerification(link, context)
		if err != nil {
			log.Printf("could not check verification for link %s: %v", link, err)
			continue
		}

		if logs {
			if isVerified {
				links = append(links, link)
				fmt.Printf("✓ Verified: %s\n", link)
			} else {
				fmt.Printf("✗ Unverified: %s\n", link)
			}
		}
	}

	return links

}

func search(url string, logs bool) {
	fmt.Printf("Searching for verified items on %s...\n", url)
	links := getLinks(url, logs)
	if len(links) == 0 {
		fmt.Println("No verified links found.")
	}
	fmt.Println("\n\nVerified links:\n")
	for _, link := range links {
		fmt.Println(link)
	}
}

func main() {
	cmd := &cli.Command{
		Name:  "vinted",
		Usage: "Search for verified Vinted items",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "search",
				Usage: "Item to search",
			},
			&cli.BoolFlag{
				Name:  "no-logs",
				Usage: "Enable logs",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			search_text := cmd.String("search")
			if search_text == "" {
				return fmt.Errorf("search is required: use --search [Item_name]")
			}
			url := fmt.Sprintf("https://www.vinted.pl/catalog?search_text=%s", search_text)
			search(url, !cmd.Bool("logs"))
			return nil
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
