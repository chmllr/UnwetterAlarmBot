package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/yhat/scrape"
	"golang.org/x/net/html"
)

const (
	url         = "http://alarm.meteocentrale.ch/getwarning_de.php?plz=%s&uwz=UWZ-CH&lang=de"
	testBaseUrl = "http://localhost:7070"
	interval    = 3 * time.Hour
)

type PLZ string
type subscriberID int

var plz2subs = map[PLZ][]subscriberID{}
var subscribers = map[subscriberID]struct{}{}
var plzRE = regexp.MustCompile(`\d\d\d\d`)

func main() {
	bot, err := tgbotapi.NewBotAPI(os.Getenv("TOKEN"))
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = os.Getenv("DEBUG_MODE") != ""

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

		// errMsg := "Ein Fehler. Bitte noch mal versuchen!"
		msg := "NOT IMPLLEMENTED YET LOL"
		inMsg := update.Message.Text
		if strings.Contains(inMsg, "abmelden") {
			msg = unregisteredMessage()
		} else if plzRE.MatchString(inMsg) {
			msg = registeredMessage(inMsg)
		} else {
			msg = startMessage(update.Message.From.FirstName)
		}

		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, msg))
	}
	// ws, err := fetch("5621")
	// if err != nil {
	// 	panic(err.Error())
	// }

	// data, err := json.Marshal(ws)
	// if err != nil {
	// 	panic(err.Error())
	// }
	// fmt.Println(string(data))
}

func fetch(plz string) ([]*warning, error) {
	effUrl := url
	if os.Getenv("DEBUG_MODE") != "" {
		effUrl = fmt.Sprintf("%s/test_page_%%s_%s.html", testBaseUrl, os.Getenv("PAGE"))
	}
	resp, err := http.Get(fmt.Sprintf(effUrl, plz))
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
	Title, Issued string
	Text          []string
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
	if strings.Contains(text, "keine Warnung aktiv") {
		return nil, nil
	}

	re := regexp.MustCompile(`(?m)^\(\d+\)\n((.|\s)*?zuletzt aktualisiert)`)
	items := re.FindAllString(text, -1)

	warnings := []*warning{}
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
		warnings = append(warnings, &warning{
			Title:  title,
			Text:   itemLines[textStart+1 : l],
			Issued: itemLines[l]})
	}

	return warnings, nil
}

func startMessage(name string) string {
	return `Hello ` + name + `!
Dieser Bot liefert Unwetterwarnungen für die Schweiz.

Bitte gebe Deine Postleitzahl ein.`
}

func registeredMessage(plz string) string {
	return `Du bist jetzt für die PLZ "` + plz + `" registriert!
	
Um sich abzumelden, einfach die Nachricht "abmelden" schicken.`
}

func unregisteredMessage() string {
	return "Du bist abgemeldet."
}
