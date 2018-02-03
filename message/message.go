package message

import "fmt"

func Start(name string) string {
	return `Hello ` + name + `!
Dieser Bot liefert Unwetterwarnungen für die Schweiz.

Bitte gebe Deine Postleitzahl ein.`
}

func Registered(plz string) string {
	return `Du wirst ab jetzt alle Unwetterwarnungen für die PLZ "` + plz + `" von mir erhalten!
	
Um sich abzumelden, einfach die Nachricht "abmelden" schicken.`
}

func Unregistered(plzs int) string {
	return fmt.Sprintf("Du wurdest von %d PLZ'en abgemeldet.", plzs)
}

const Error = "Oops, ein Fehler!"
