package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gorilla/feeds"
	"github.com/yhat/scrape"
	"golang.org/x/net/html"
)

const (
	url      = "http://alarm.meteocentrale.ch/getwarning_de.php?plz=5621&uwz=UWZ-CH&lang=de"
	testUrl  = "http://localhost:7070/"
	interval = 3 * time.Hour
)

var (
	lastWarning *warning
	rss         atomic.Value
	lastContent string
)

func handler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "%s", rss.Load().(string))
}

func main() {
	go func() {
		for {
			start := time.Now()
			currentWarning, err := fetch()
			if err != nil {
				log.Println("fetch failed:", err)
			} else if !currentWarning.Equal(lastWarning) {
				log.Println("new warning!")
				lastWarning = currentWarning
				if rssFeed, err := warning2RSS(currentWarning); err != nil {
					log.Println("rss rendering failed:", err)
				} else {
					rss.Store(rssFeed)
				}
			} else {
				log.Println("warnings identical; skipping...")
			}
			log.Println("request took", time.Since(start))
			time.Sleep(interval)
		}
	}()
	if rssFeed, err := warning2RSS(nil); err != nil {
		panic("couldn't render empty rss feed: " + err.Error())
	} else {
		rss.Store(rssFeed)
	}
	http.HandleFunc("/", handler)
	http.ListenAndServe(":"+os.Getenv("PORT"), nil)
}

func warning2RSS(w *warning) (string, error) {
	feed := &feeds.Feed{
		Title:       "Unwetter Warnung",
		Link:        &feeds.Link{Href: "https://github.com/chmllr/nepogoda"},
		Description: "Unwetter Warnung für die Schweiz",
		Author:      &feeds.Author{Name: "Christian Müller", Email: "@drmllr"},
		Created:     time.Now(),
	}
	if w != nil {
		feed.Items = []*feeds.Item{
			{
				Title:       w.title,
				Link:        &feeds.Link{Href: fmt.Sprintf("%s&fetched=%d", url, w.fetched.Unix())},
				Description: w.text,
				Created:     w.fetched,
			},
		}
	}
	return feed.ToRss()
}

func fetch() (*warning, error) {
	effUrl := url
	if os.Getenv("TEST") != "" {
		effUrl = testUrl + os.Getenv("PAGE")
	}
	resp, err := http.Get(effUrl)
	if err != nil {
		return nil, fmt.Errorf("couldn't fetch page: %v", err)
	}
	defer resp.Body.Close()

	root, err := html.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("couldn't parse response: %v", err)
	}
	content, ok := scrape.Find(root, scrape.ById("content"))
	if ok {
		return getWarning(content)
	}

	return nil, fmt.Errorf("couldn't find content")
}

type warning struct {
	title   string
	text    string
	fetched time.Time
}

func (w *warning) Equal(other *warning) bool {
	return other != nil && w.title == other.title && w.text == other.text
}

func getWarning(node *html.Node) (*warning, error) {
	text := scrape.TextJoin(node, func(allLines []string) string {
		lines := []string{}
		for _, line := range allLines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				lines = append(lines, trimmed)
			}
		}
		text := strings.Join(lines, "\n")
		re := regexp.MustCompile(`(?m)^Unwetterwarnungen.*\n((.|\s)*)Die Höhen`)
		matches := re.FindAllStringSubmatch(text, -1)
		if len(matches) < 1 || len(matches[0]) < 2 {
			return "PARSE_ERROR"
		}
		text = matches[0][1]

		oldContent := lastContent
		lastContent = text
		if lastContent == oldContent || strings.Contains(text, "keine Warnung aktiv") {
			return ""
		}

		re = regexp.MustCompile(`(?m)^\(\d+\)\n((.|\s)*?zuletzt aktualisiert)`)
		items := re.FindAllString(text, -1)

		warnings := []string{}
		for _, item := range items {
			re = regexp.MustCompile(`(?m)^(gültig) (.*)\s+(.*)$`)
			item = re.ReplaceAllString(item, `$1 $2 <b>$3</b>`)
			itemLines := strings.Split(item, "\n")
			itemLines[1] = "<b>" + itemLines[1] + "</b>"
			l := len(itemLines) - 1
			itemLines[l] = "<br><i>" + itemLines[l] + "</i>"
			warnings = append(warnings, strings.Join(itemLines[1:], "<br>\n"))
		}
		return strings.Join(warnings, "<br><br>\n")
	})
	switch text {
	case "":
		return nil, fmt.Errorf("no warning found")
	case "PARSE_ERROR":
		return &warning{"Neue Unwetterwarnung!", url, time.Now()}, nil
	}
	return &warning{"Neue Unwetterwarnung!", text, time.Now()}, nil
}
