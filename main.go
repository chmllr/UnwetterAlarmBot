package main

import (
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/chmllr/nepogoda/message"
	"github.com/chmllr/nepogoda/scrape"
	"github.com/chmllr/nepogoda/storage"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

var (
	plzRE = regexp.MustCompile(`\d\d\d\d`)
)

func main() {
	vol := storage.Volume{}
	cache := map[string]string{}

	warningStream := make(chan *scrape.PLZWarnings)
	go scrape.FetchLoop(warningStream, 5*time.Second, vol)

	bot, err := tgbotapi.NewBotAPI(os.Getenv("TOKEN"))
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = os.Getenv("DEBUG_MODE") != ""

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		panic(err.Error())
	}

	for {
		select {
		case warning := <-warningStream:
			subscribers := vol.Subscribers(warning.PLZ)
			hash := ""
			for _, w := range warning.Warnings {
				hash = hash + w.Hash()
			}
			if cache[warning.PLZ] == hash {
				continue
			}
			cache[warning.PLZ] = hash
			for _, s := range subscribers {
				for _, w := range warning.Warnings {
					msg := tgbotapi.NewMessage(s.ChatID, w.String())
					msg.ParseMode = "Markdown"
					bot.Send(msg)
				}
			}
		case update := <-updates:
			if update.Message == nil {
				break
			}

			log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

			var msg string
			userID := update.Message.From.ID
			inMsg := update.Message.Text
			if plzRE.MatchString(inMsg) {
				if err := vol.Register(userID, update.Message.Chat.ID, inMsg); err != nil {
					log.Println(err)
					msg = "Ich konnte dich nicht für diese PLZ anmelden!"
				} else {
					msg = message.Registered(inMsg)
				}
			} else if strings.Contains(inMsg, "abmelden") {
				plzs := vol.Unregister(userID)
				msg = message.Unregistered(plzs)
			} else {
				msg = message.Start(update.Message.From.FirstName)
			}

			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, msg))
		}
	}

}
