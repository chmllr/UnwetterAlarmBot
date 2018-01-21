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
			warnings, err := fetch()
			if err != nil {
				log.Println("fetch failed:", err)
			} else {
				log.Println("new warning!")
				if rssFeed, err := warnings2RSS(warnings); err != nil {
					log.Println("rss rendering failed:", err)
				} else {
					rss.Store(rssFeed)
				}
			}
			log.Println("request took", time.Since(start))
			time.Sleep(interval)
		}
	}()
	if rssFeed, err := warnings2RSS(nil); err != nil {
		panic("couldn't render empty rss feed: " + err.Error())
	} else {
		rss.Store(rssFeed)
	}
	http.HandleFunc("/", handler)
	http.ListenAndServe(":"+os.Getenv("PORT"), nil)
}

func warnings2RSS(ws []*warning) (string, error) {
	feed := &feeds.Feed{
		Title:       "Unwetter Warnung",
		Link:        &feeds.Link{Href: "https://github.com/chmllr/nepogoda"},
		Description: "Unwetter Warnung für die Schweiz",
		Author:      &feeds.Author{Name: "Christian Müller", Email: "@drmllr"},
		Created:     time.Now(),
	}
	for _, w := range ws {
		feed.Items = append(feed.Items, &feeds.Item{
			Title:       w.title,
			Link:        &feeds.Link{Href: fmt.Sprintf("%s&fetched=%d", url, w.fetched.Unix())},
			Description: fmt.Sprintf("%s<br/><br/><i>%s</i>", w.text, w.issued),
			Created:     w.fetched,
		})
	}
	return feed.ToRss()
}

func fetch() ([]*warning, error) {
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
		return getWarnings(content)
	}

	return nil, fmt.Errorf("couldn't find content")
}

type warning struct {
	title, text, issued string
	fetched             time.Time
}

func getWarnings(node *html.Node) ([]*warning, error) {
	text := scrape.TextJoin(node, func(ls []string) string { return strings.Join(ls, "\n") })

	lines := []string{}
	skip := true
	for _, line := range strings.Split(text, "\n") {
		if strings.Contains(line, "Unwetterwarnungen") {
			skip = false
			continue
		}
		if skip {
			continue
		}
		if strings.Contains(line, "Die Höhenstufen des Bereichs") {
			break
		}
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			lines = append(lines, trimmed)
		}
	}

	text = strings.Join(lines, "\n")
	oldContent := lastContent
	lastContent = text
	if lastContent == oldContent || strings.Contains(text, "keine Warnung aktiv") {
		return nil, fmt.Errorf("no warning found")
	}

	re := regexp.MustCompile(`(?m)^\(\d+\)\n((.|\s)*?zuletzt aktualisiert)`)
	items := re.FindAllString(text, -1)

	warnings := []*warning{}
	for _, item := range items {
		re = regexp.MustCompile(`(?m)^gültig für:\s+?(.*)$`)
		item = re.ReplaceAllString(item, "")
		re = regexp.MustCompile(`(?m)^(gültig) (.*)\s+(.*)$`)
		item = re.ReplaceAllString(item, `$1 $2 <b>$3</b>`)
		itemLines := strings.Split(item, "\n")
		title := ""
		textStart := 0
		for k, v := range itemLines[1:] {
			if strings.Contains(v, "gültig") {
				textStart = k
				break
			}
			title += v + " "
		}
		l := len(itemLines) - 1
		warnings = append(warnings, &warning{
			title:  title,
			text:   strings.Join(itemLines[textStart+1:l], "<br>\n"),
			issued: itemLines[l]})
	}

	return warnings, nil
}
