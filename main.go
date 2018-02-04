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
	if err := vol.Load("volume.json"); err != nil {
		panic(err.Error())
	}

	cache := storage.Cache{}
	if err := cache.Load("cache.json"); err != nil {
		panic(err.Error())
	}

	warnings := make(chan *scrape.PLZWarnings)
	go scrape.FetchLoop(warnings, 2*time.Hour, vol)

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
		case update := <-warnings:
			if len(update.Warnings) == 0 {
				cache.Clear(update.PLZ)
				break
			}
			subscribers := vol.Subscribers(update.PLZ)
			for _, w := range update.Warnings {
				hash := w.Hash()
				if cache.Has(update.PLZ, hash) {
					continue
				}
				cache.Set(update.PLZ, hash)
				text := w.String()
				for _, s := range subscribers {
					msg := tgbotapi.NewMessage(s.ChatID, text)
					msg.ParseMode = "Markdown"
					bot.Send(msg)
				}
			}
		case update := <-updates:
			if update.Message == nil {
				break
			}
			log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)
			msg := message.Start(update.Message.From.FirstName)
			userID := update.Message.From.ID
			inMsg := update.Message.Text
			if plzRE.MatchString(inMsg) {
				msg = message.Registered(inMsg)
				if err := vol.Register(userID, update.Message.Chat.ID, inMsg); err != nil {
					log.Println(err)
					msg = "Ich konnte dich nicht für diese PLZ anmelden!"
				}
			} else if strings.Contains(inMsg, "abmelden") {
				plzs, err := vol.Unregister(userID)
				msg = message.Unregistered(plzs)
				if err != nil {
					msg = message.Error
				}
			}
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, msg))
		}
	}

}
