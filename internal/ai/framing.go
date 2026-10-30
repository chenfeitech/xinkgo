package ai

import (
	"bufio"
	"io"
	"strings"
)

// maxPieceBytes is the longest one piece of an answer that is read. A
// longer one is a mistake, and would only fill the memory.
const maxPieceBytes = 4 << 20

// ndjsonPieces reads an answer that is one JSON object to a line, and hands
// each line on as it comes. Empty lines are passed over.
func ndjsonPieces(r io.Reader) <-chan Piece {
	out := make(chan Piece)
	go func() {
		defer close(out)
		scan := bufio.NewScanner(r)
		scan.Buffer(make([]byte, 0, 64*1024), maxPieceBytes)
		for scan.Scan() {
			line := strings.TrimSpace(scan.Text())
			if line == "" {
				continue
			}
			out <- Piece{Data: []byte(line)}
		}
		if err := scan.Err(); err != nil {
			out <- Piece{Err: err}
		}
	}()
	return out
}

// ssePieces reads an answer that is sent as server-sent events, and hands
// on each event's `data:` as it comes: the several `data:` lines of one
// event are one piece, joined by the newlines that parted them.
//
// Comments — the lines that begin with a colon, sent to keep the line open
// — are passed over, as is every field that is not data.
func ssePieces(r io.Reader) <-chan Piece {
	out := make(chan Piece)
	go func() {
		defer close(out)
		scan := bufio.NewScanner(r)
		scan.Buffer(make([]byte, 0, 64*1024), maxPieceBytes)
		var event []string
		send := func() {
			if len(event) == 0 {
				return
			}
			out <- Piece{Data: []byte(strings.Join(event, "\n"))}
			event = nil
		}
		for scan.Scan() {
			line := strings.TrimRight(scan.Text(), "\r")
			if line == "" {
				// The blank line is the end of the event.
				send()
				continue
			}
			if strings.HasPrefix(line, ":") {
				continue
			}
			name, value, found := strings.Cut(line, ":")
			if !found || name != "data" {
				continue
			}
			event = append(event, strings.TrimPrefix(value, " "))
		}
		// An answer that stops without a blank line at the end is still
		// an answer.
		send()
		if err := scan.Err(); err != nil {
			out <- Piece{Err: err}
		}
	}()
	return out
}
