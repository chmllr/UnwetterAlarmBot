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
	interval = 6 * time.Hour
)

var warnungIssued time.Time
var lastWarning string
var rss atomic.Value

func handler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "%s", rss.Load().(string))
}

func main() {
	go func() {
		for {
			title, text, err := fetch()
			if err != nil {
				log.Println(err)
			}
			current := title + strings.Join(text, "\n")
			if current != lastWarning {
				log.Println("new warning!")
				lastWarning = current
				warnungIssued = time.Now()
				start := time.Now()
				if rssFeed, err := warning2RSS(title, text); err != nil {
					log.Println(err)
				} else {
					rss.Store(rssFeed)
				}
				log.Println("request took", time.Since(start))
			} else {
				log.Println("warning identical; skipping...")
			}
			time.Sleep(interval)
		}
	}()
	http.HandleFunc("/", handler)
	http.ListenAndServe(":"+os.Getenv("PORT"), nil)
}

func warning2RSS(title string, text []string) (string, error) {
	feed := &feeds.Feed{
		Title:       "Unwetter Warnung",
		Link:        &feeds.Link{Href: "https://github.com/chmllr/nepogoda"},
		Description: "Unwetter Warnung für die Schweiz",
		Author:      &feeds.Author{Name: "Christian Müller", Email: "@drmllr"},
		Created:     time.Now(),
	}

	feed.Items = []*feeds.Item{
		{
			Title:       title,
			Link:        &feeds.Link{Href: url},
			Description: strings.Join(text, "<br>"),
			Created:     warnungIssued,
		},
	}

	return feed.ToRss()
}

func fetch() (string, []string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", nil, fmt.Errorf("couldn't fetch page: %v", err)
	}
	defer resp.Body.Close()

	root, err := html.Parse(resp.Body)
	if err != nil {
		return "", nil, fmt.Errorf("couldn't parse response: %v", err)
	}
	content, ok := scrape.Find(root, scrape.ById("content"))
	if ok {
		title, text := getText(content)
		return title, text, nil
	}

	return "", nil, fmt.Errorf("couldn't find content")
}

func getText(node *html.Node) (string, []string) {
	text := scrape.TextJoin(node, func(allLines []string) string {
		lines := []string{}
		for _, line := range allLines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				lines = append(lines, trimmed)
			}
		}
		text := strings.Join(lines, "\n")
		re := regexp.MustCompile(`(?m)^(Unwetterwarnung Stufe(.|\s)*)Die Höhen`)
		text = re.FindAllStringSubmatch(text, -1)[0][1]
		re = regexp.MustCompile(`(?m)^(gültig) (.*)\s+(.*)$`)
		return re.ReplaceAllString(text, `$1 $2 <b>$3</b>`)
	})
	lines := strings.Split(strings.TrimSpace(text), "\n")
	lines[len(lines)-2] = "<br>" + lines[len(lines)-2]
	lines[len(lines)-1] = "<br><i>" + lines[len(lines)-1] + "</i>"
	return lines[0], lines[1:]
}
