// Package ics writes iCalendar (RFC 5545) files, which Apple Calendar,
// Google Calendar and Outlook import or subscribe to.
package ics

import (
	"strings"
	"time"
)

type Event struct {
	UID         string
	Start, End  time.Time
	Summary     string
	Location    string
	Description string
	Updated     time.Time
}

// Calendar renders the events as one calendar. Times are written in UTC, so
// no VTIMEZONE block is needed and every app shows them in its own zone.
func Calendar(name string, events []Event) []byte {
	var b strings.Builder
	line := func(s string) { fold(&b, s) }
	line("BEGIN:VCALENDAR")
	line("VERSION:2.0")
	line("PRODID:-//GMIM Siloam//Jadwal Ibadah//ID")
	line("CALSCALE:GREGORIAN")
	line("METHOD:PUBLISH")
	line("X-WR-CALNAME:" + escape(name))
	line("REFRESH-INTERVAL;VALUE=DURATION:PT6H")
	line("X-PUBLISHED-TTL:PT6H")
	for _, e := range events {
		stamp := e.Updated
		if stamp.IsZero() {
			stamp = time.Now()
		}
		line("BEGIN:VEVENT")
		line("UID:" + e.UID)
		line("DTSTAMP:" + utc(stamp))
		line("DTSTART:" + utc(e.Start))
		line("DTEND:" + utc(e.End))
		line("SUMMARY:" + escape(e.Summary))
		if e.Location != "" {
			line("LOCATION:" + escape(e.Location))
		}
		if e.Description != "" {
			line("DESCRIPTION:" + escape(e.Description))
		}
		line("END:VEVENT")
	}
	line("END:VCALENDAR")
	return []byte(b.String())
}

func utc(t time.Time) string { return t.UTC().Format("20060102T150405Z") }

// escape applies the TEXT escaping rules: backslash, semicolon, comma, newline.
func escape(s string) string {
	return strings.NewReplacer(`\`, `\\`, ";", `\;`, ",", `\,`, "\r\n", `\n`, "\n", `\n`).Replace(s)
}

// fold writes one content line, folded at 75 octets without splitting a
// UTF-8 character, ending in CRLF.
func fold(b *strings.Builder, s string) {
	limit := 75
	for len(s) > limit {
		cut := limit
		for cut > 0 && !startsRune(s[cut]) {
			cut--
		}
		b.WriteString(s[:cut])
		b.WriteString("\r\n ")
		s = s[cut:]
		limit = 74 // continuation lines start with a space
	}
	b.WriteString(s)
	b.WriteString("\r\n")
}

func startsRune(c byte) bool { return c&0xC0 != 0x80 }
