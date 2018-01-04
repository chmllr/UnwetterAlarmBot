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
			warn, err := fetch()
			if err != nil {
				log.Println(err)
			}
			current := warn.title + strings.Join(warn.text, "\n")
			if current != lastWarning {
				log.Println("new warning!")
				lastWarning = current
				warnungIssued = time.Now()
				start := time.Now()
				if rssFeed, err := warning2RSS(warn); err != nil {
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

func warning2RSS(w *warning) (string, error) {
	feed := &feeds.Feed{
		Title:       "Unwetter Warnung",
		Link:        &feeds.Link{Href: "https://github.com/chmllr/nepogoda"},
		Description: "Unwetter Warnung für die Schweiz",
		Author:      &feeds.Author{Name: "Christian Müller", Email: "@drmllr"},
		Created:     time.Now(),
	}

	feed.Items = []*feeds.Item{
		{
			Title:       w.title,
			Link:        &feeds.Link{Href: fmt.Sprintf("%s&unixTime=%d", url, warnungIssued.Unix())},
			Description: strings.Join(w.text, "<br>"),
			Created:     warnungIssued,
		},
	}

	return feed.ToRss()
}

func fetch() (*warning, error) {
	resp, err := http.Get(url)
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
	title string
	text  []string
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
		re := regexp.MustCompile(`(?m)^(Unwetterwarnung Stufe(.|\s)*)Die Höhen`)
		matches := re.FindAllStringSubmatch(text, -1)
		if len(matches) < 1 || len(matches[0]) < 2 {
			return ""
		}
		text = matches[0][1]
		re = regexp.MustCompile(`(?m)^(gültig) (.*)\s+(.*)$`)
		return re.ReplaceAllString(text, `$1 $2 <b>$3</b>`)
	})
	if text == "" {
		return nil, fmt.Errorf("no warning found")
	}
	lines := strings.Split(strings.TrimSpace(text), "\n")
	lines[len(lines)-2] = "<br>" + lines[len(lines)-2]
	lines[len(lines)-1] = "<br><i>" + lines[len(lines)-1] + "</i>"
	return &warning{lines[0], lines[1:]}, nil
}
