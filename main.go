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
				Description: strings.Join(w.text, "<br>"),
				Created:     w.fetched,
			},
		}
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
	title   string
	text    []string
	fetched time.Time
}

func (w *warning) Equal(other *warning) bool {
	return other != nil && w.title == other.title &&
		strings.Join(w.text, "") == strings.Join(other.text, "")
}

func getWarning(node *html.Node) (*warning, error) {
	text := scrape.TextJoin(node, func(allLines []string) string {
		lines := []string{}
		for _, line := range allLines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				lines = append(lines, trimmed)
			}
			if strings.Contains(line, "Details zu den Warnstufen") {
				break
			}
		}
		text := strings.Join(lines, "\n")
		oldContent := lastContent
		lastContent = text
		if lastContent == oldContent || strings.Contains(text, "keine Warnung aktiv") {
			return ""
		}
		re := regexp.MustCompile(`(?m)^(Unwetterwarnung Stufe(.|\s)*)Die Höhen`)
		matches := re.FindAllStringSubmatch(text, -1)
		if len(matches) < 1 || len(matches[0]) < 2 {
			return "PARSE_ERROR"
		}
		text = matches[0][1]
		re = regexp.MustCompile(`(?m)^(gültig) (.*)\s+(.*)$`)
		return re.ReplaceAllString(text, `$1 $2 <b>$3</b>`)
	})
	switch text {
	case "":
		return nil, fmt.Errorf("no warning found")
	case "PARSE_ERROR":
		return &warning{"Neue Wetterwarnung!", []string{url}, time.Now()}, nil
	}

	lines := strings.Split(strings.TrimSpace(text), "\n")
	lines[len(lines)-2] = "<br>" + lines[len(lines)-2]
	lines[len(lines)-1] = "<br><i>" + lines[len(lines)-1] + "</i>"
	return &warning{lines[0], lines[1:], time.Now()}, nil
}
