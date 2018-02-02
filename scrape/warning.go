package scrape

import (
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/chmllr/nepogoda/storage"
	"github.com/yhat/scrape"
	"golang.org/x/net/html"
)

const (
	url         = "http://alarm.meteocentrale.ch/getwarning_de.php?plz=%s&uwz=UWZ-CH&lang=de"
	testBaseUrl = "http://localhost:7070"
	interval    = 2 * time.Hour
)

type PLZWarnings struct {
	PLZ      string
	Warnings []*Warning
}

func FetchLoop(stream chan *PLZWarnings, t time.Duration, vol storage.Volume) {
	for {
		start := time.Now()
		plzs := vol.PLZs()
		for _, plz := range plzs {
			ws, err := fetch(plz)
			if err != nil {
				log.Println(err)
				continue
			}
			stream <- &PLZWarnings{plz, ws}
		}
		log.Printf("fetched %d PLZs in %s\n", len(plzs), time.Since(start))
		time.Sleep(t)
	}
}

func fetch(plz string) ([]*Warning, error) {
	effUrl := url
	if os.Getenv("DEBUG_MODE") != "" {
		effUrl = fmt.Sprintf("%s/test_page_%%s_%s.html", testBaseUrl, os.Getenv("PAGE"))
	}
	resp, err := http.Get(fmt.Sprintf(effUrl, plz))
	if err != nil {
		return nil, fmt.Errorf("couldn't fetch page: %v", err)
	}
	defer resp.Body.Close()
	return scrapeWarnings(resp.Body)
}

type Warning struct {
	Title, Issued string
	Text          []string
}

func (w *Warning) String() string {
	return fmt.Sprintf("*%s*\n\n%s\n\n_%s_", w.Title, strings.Join(w.Text, "\n"), w.Issued)
}

func (w *Warning) Hash() string {
	return fmt.Sprintf("%x", md5.Sum([]byte(w.String())))
}

func scrapeWarnings(body io.Reader) ([]*Warning, error) {
	root, err := html.Parse(body)
	if err != nil {
		return nil, fmt.Errorf("couldn't parse response: %v", err)
	}
	node, ok := scrape.Find(root, scrape.ById("content"))
	if !ok {
		return nil, fmt.Errorf("couldn't find content")
	}

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
	if strings.Contains(text, "keine Warnung aktiv") {
		return nil, nil
	}

	re := regexp.MustCompile(`(?m)^\(\d+\)\n((.|\s)*?zuletzt aktualisiert)`)
	items := re.FindAllString(text, -1)

	warnings := []*Warning{}
	for _, item := range items {
		re = regexp.MustCompile(`(?m)^gültig für:\s+?(.*)$`)
		item = re.ReplaceAllString(item, "")
		re = regexp.MustCompile(`(?m)^(gültig) (.*)\s+(.*)$`)
		item = re.ReplaceAllString(item, `$1 $2 *$3*`)
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
		warnings = append(warnings, &Warning{
			Title:  title,
			Text:   itemLines[textStart+1 : l],
			Issued: itemLines[l]})
	}

	return warnings, nil
}
