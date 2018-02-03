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
	err := vol.Load("volume.json")
	if err != nil {
		panic(err.Error())
	}

	cache := storage.Cache{}
	err = cache.Load("cache.json")
	if err != nil {
		panic(err.Error())
	}

	warningsStream := make(chan *scrape.PLZWarnings)
	go scrape.FetchLoop(warningsStream, 5*time.Second, vol)

	bot, err := tgbotapi.NewBotAPI(os.Getenv("TOKEN"))
	if err != nil {
		log.Panic(err)
	}

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		panic(err.Error())
	}

	for {
		select {
		case warnings := <-warningsStream:
			if len(warnings.Warnings) == 0 {
				cache.Clear(warnings.PLZ)
				break
			}
			subscribers := vol.Subscribers(warnings.PLZ)
			for _, w := range warnings.Warnings {
				hash := w.Hash()
				if cache.Has(warnings.PLZ, hash) {
					continue
				}
				cache.Set(warnings.PLZ, hash)
				for _, s := range subscribers {
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
				plzs, err := vol.Unregister(userID)
				msg = message.Unregistered(plzs)
				if err != nil {
					msg = message.Error()
				}

			} else {
				msg = message.Start(update.Message.From.FirstName)
			}

			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, msg))
		}
	}

}
