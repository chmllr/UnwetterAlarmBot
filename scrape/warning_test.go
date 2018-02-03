package scrape

import (
	"bytes"
	"io/ioutil"
	"testing"
)

func verify(t *testing.T, file string, expWs []string) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		t.Fatal(err.Error())
	}

	ws, err := scrapeWarnings(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err.Error())
	}

	for i, w := range ws {
		if w.String() != expWs[i] {
			t.Fatalf("warning %d fetched from %q differes from the expected one\n=== Want: \n%s\n=== Got:\n%s",
				i+1, file, expWs[i], w)
		}
	}
}

func TestFile1(t *testing.T) {
	verify(t, "../test_page_5621_1.html", []string{`*Unwetterwarnung Stufe  Orange vor Sturm/Orkan*

*gültig von*: Montag, 15. Januar 2018, 20:00 Uhr
*gültig bis*: Donnerstag, 18. Januar 2018, 00:00 Uhr

Ab Montagabend und -nacht ist zeitweise kräftiger Wind möglich. Dabei kommt es zu Sturmböen zwischen 70 und 100 km/h, örtlich auch mehr. Der Wind weht aus Südwest bis West. Mittwochnacht lässt der Wind vorübergehend nach.

_Diese Warnung wurde am Montag, 15. Januar 2018, 12:18 Uhr zuletzt aktualisiert_`})
}

func TestFile2(t *testing.T) {
	verify(t, "../test_page_5621_2.html", []string{
		`*Unwetterwarnung Stufe  Orange vor Sturm/Orkan*

*gültig von*: Montag, 15. Januar 2018, 20:00 Uhr
*gültig bis*: Donnerstag, 18. Januar 2018, 00:00 Uhr

Ab Montagabend und -nacht ist zeitweise kräftiger Wind möglich. Dabei kommt es zu Sturmböen zwischen 70 und 100 km/h, örtlich auch mehr. Der Wind weht aus Südwest bis West. Mittwochnacht lässt der Wind vorübergehend nach.

_Diese Warnung wurde am Montag, 15. Januar 2018, 12:18 Uhr zuletzt aktualisiert_`,
		`*Vorwarnung vor Gewitter, Warnstufe Orange möglich*

*gültig von*: Mittwoch, 17. Januar 2018, 00:00 Uhr
*gültig bis*: Mittwoch, 17. Januar 2018, 06:00 Uhr

Dienstagnacht sind lokal Wintergewitter möglich, verbunden mit Starkregen, Sturmböen und dann Schneefall bis ins Flachland.

_Diese Vorwarnung wurde am Dienstag, 16. Januar 2018, 18:37 Uhr zuletzt aktualisiert_`,
	})
}
