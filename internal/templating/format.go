// Package templating turns a Moment-style date format into a note name and
// expands the template variables Xinkgo's core plugins understand. It is
// pure: no I/O, no windows, no dependencies beyond the standard library.
//
// Only a small, documented subset of Moment's tokens is accepted. A token
// the package does not know is an error, never a silently wrong file name.
package templating

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// The formats a first start uses.
const (
	DefaultDateFormat = "YYYY-MM-DD"
	DefaultTimeFormat = "HH:mm"
)

var monthNames = []string{
	"January", "February", "March", "April", "May", "June",
	"July", "August", "September", "October", "November", "December",
}
var monthShort = []string{
	"Jan", "Feb", "Mar", "Apr", "May", "Jun",
	"Jul", "Aug", "Sep", "Oct", "Nov", "Dec",
}
var dayNames = []string{
	"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday",
	"Saturday",
}
var dayShort = []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}

// hour12 is the hour on a twelve-hour clock, 1 to 12.
func hour12(t time.Time) int {
	h := t.Hour() % 12
	if h == 0 {
		return 12
	}
	return h
}

// token is one format token. Name is the Moment spelling, Re the regexp a
// name must match, and Emit turns a time into that piece of the name.
type token struct {
	Name string
	Re   string
	Emit func(time.Time) string
}

// tokens is longest-first within each prefix, so MMMM wins over MM.
var tokens = []token{
	{"YYYY", `(\d{4})`, func(t time.Time) string { return t.Format("2006") }},
	{"YY", `(\d{2})`, func(t time.Time) string { return t.Format("06") }},
	{"MMMM", `([A-Za-z]+)`, func(t time.Time) string { return t.Month().String() }},
	{"MMM", `([A-Za-z]+)`, func(t time.Time) string { return t.Format("Jan") }},
	{"MM", `(\d{2})`, func(t time.Time) string { return t.Format("01") }},
	{"M", `(\d{1,2})`, func(t time.Time) string { return strconv.Itoa(int(t.Month())) }},
	{"DD", `(\d{2})`, func(t time.Time) string { return t.Format("02") }},
	{"D", `(\d{1,2})`, func(t time.Time) string { return strconv.Itoa(t.Day()) }},
	{"dddd", `([A-Za-z]+)`, func(t time.Time) string { return t.Weekday().String() }},
	{"ddd", `([A-Za-z]+)`, func(t time.Time) string { return t.Format("Mon") }},
	{"HH", `(\d{2})`, func(t time.Time) string { return t.Format("15") }},
	{"H", `(\d{1,2})`, func(t time.Time) string { return strconv.Itoa(t.Hour()) }},
	{"hh", `(\d{2})`, func(t time.Time) string { return t.Format("03") }},
	{"h", `(\d{1,2})`, func(t time.Time) string { return strconv.Itoa(hour12(t)) }},
	{"mm", `(\d{2})`, func(t time.Time) string { return t.Format("04") }},
	{"ss", `(\d{2})`, func(t time.Time) string { return t.Format("05") }},
	{"A", `(AM|PM)`, func(t time.Time) string { return t.Format("PM") }},
	{"a", `(am|pm)`, func(t time.Time) string { return t.Format("pm") }},
}

func tokenNamed(name string) (token, bool) {
	for _, one := range tokens {
		if one.Name == name {
			return one, true
		}
	}
	return token{}, false
}

// piece is one part of a compiled format: either a literal or a token.
type piece struct {
	Literal string
	Token   string
}

// compile walks a format once. Brackets [like this] hold a literal; any
// ASCII letter outside a token is refused, since Moment has more tokens
// than this package implements and guessing would produce wrong names.
func compile(format string) ([]piece, error) {
	if strings.TrimSpace(format) == "" {
		return nil, errors.New("empty format")
	}
	var out []piece
	for i := 0; i < len(format); {
		if format[i] == '[' {
			end := strings.IndexByte(format[i+1:], ']')
			if end < 0 {
				return nil, errors.New("unclosed [ in format")
			}
			out = append(out, piece{Literal: format[i+1 : i+1+end]})
			i += end + 2
			continue
		}
		matched := ""
		for _, one := range tokens {
			if strings.HasPrefix(format[i:], one.Name) {
				matched = one.Name
				break
			}
		}
		if matched != "" {
			out = append(out, piece{Token: matched})
			i += len(matched)
			continue
		}
		b := format[i]
		if isASCIILetter(b) {
			return nil, fmt.Errorf("unsupported token near %q", format[i:])
		}
		j := i
		for j < len(format) {
			if format[j] == '[' || isASCIILetter(format[j]) {
				break
			}
			j++
		}
		out = append(out, piece{Literal: format[i:j]})
		i = j
	}
	return out, nil
}

func isASCIILetter(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z'
}

// Validate reports whether a format can be used to name a note.
func Validate(format string) error {
	_, err := compile(format)
	return err
}

// Format writes t in the given Moment-style format.
func Format(t time.Time, format string) (string, error) {
	pieces, err := compile(format)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, one := range pieces {
		if one.Token == "" {
			b.WriteString(one.Literal)
			continue
		}
		tok, _ := tokenNamed(one.Token)
		b.WriteString(tok.Emit(t))
	}
	return b.String(), nil
}

type parsedDate struct {
	year, month, day, hour, min, sec int
	hasYear, hasMonth, hasDay        bool
	twelveHour                       bool
	pm                               *bool
}

// Parse is the inverse of Format. A format without a year, month and day
// cannot name one date, so it is refused rather than guessed.
func Parse(name, format string) (time.Time, error) {
	pieces, err := compile(format)
	if err != nil {
		return time.Time{}, err
	}
	var pattern strings.Builder
	pattern.WriteByte('^')
	var names []string
	for _, one := range pieces {
		if one.Token == "" {
			pattern.WriteString(regexp.QuoteMeta(one.Literal))
			continue
		}
		tok, _ := tokenNamed(one.Token)
		pattern.WriteString(tok.Re)
		names = append(names, one.Token)
	}
	pattern.WriteByte('$')
	rx, err := regexp.Compile(pattern.String())
	if err != nil {
		return time.Time{}, err
	}
	match := rx.FindStringSubmatch(name)
	if match == nil {
		return time.Time{}, fmt.Errorf("%q is not a note name of %q", name, format)
	}
	p := parsedDate{}
	for i, name := range names {
		if err := applyToken(&p, name, match[i+1]); err != nil {
			return time.Time{}, err
		}
	}
	if !p.hasYear || !p.hasMonth || !p.hasDay {
		return time.Time{}, errors.New("the format has no year, month and day")
	}
	if p.twelveHour && p.pm != nil {
		if *p.pm && p.hour < 12 {
			p.hour += 12
		} else if !*p.pm && p.hour == 12 {
			p.hour = 0
		}
	}
	if p.month < 1 || p.month > 12 || p.day < 1 || p.day > 31 ||
		p.hour > 23 || p.min > 59 || p.sec > 59 {
		return time.Time{}, errors.New("the name holds a date that cannot be")
	}
	got := time.Date(p.year, time.Month(p.month), p.day, p.hour, p.min,
		p.sec, 0, time.Local)
	// time.Date normalises 31 February; refuse it rather than silently
	// turning it into March.
	if got.Year() != p.year || int(got.Month()) != p.month ||
		got.Day() != p.day {
		return time.Time{}, errors.New("the name holds a date that cannot be")
	}
	return got, nil
}

func applyToken(p *parsedDate, name, value string) error {
	number := func() (int, error) { return strconv.Atoi(value) }
	switch name {
	case "YYYY", "YY":
		n, err := number()
		if err != nil {
			return err
		}
		if name == "YY" {
			if n >= 70 {
				n += 1900
			} else {
				n += 2000
			}
		}
		p.year, p.hasYear = n, true
	case "MMMM", "MMM", "MM", "M":
		list := monthNames
		if name == "MMM" {
			list = monthShort
		}
		if name == "MMMM" || name == "MMM" {
			for i, one := range list {
				if one == value {
					p.month, p.hasMonth = i+1, true
					return nil
				}
			}
			return fmt.Errorf("unknown month %q", value)
		}
		n, err := number()
		if err != nil {
			return err
		}
		p.month, p.hasMonth = n, true
	case "DD", "D":
		n, err := number()
		if err != nil {
			return err
		}
		p.day, p.hasDay = n, true
	case "dddd", "ddd":
		// The weekday is redundant; the date already says which it is.
	case "HH", "H", "hh", "h":
		n, err := number()
		if err != nil {
			return err
		}
		p.hour = n
		if name == "hh" || name == "h" {
			p.twelveHour = true
		}
	case "mm":
		n, err := number()
		if err != nil {
			return err
		}
		p.min = n
	case "ss":
		n, err := number()
		if err != nil {
			return err
		}
		p.sec = n
	case "A", "a":
		pm := value == "PM" || value == "pm" || value == "Pm"
		p.pm = &pm
	}
	return nil
}
