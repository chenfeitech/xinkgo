// Package note is one file the app has open: its text, whether it holds
// changes the disk does not, and how to write it back so that a crash
// leaves the file as it was and never half of the new one.
//
// The disk is what is true. The text here is a copy of it, plus what the
// user has typed since; every decision about reloading or saving follows
// from comparing the two.
package note

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/helightxu/xinkgo/internal/textfile"
)

// Note is a file the app has open.
type Note struct {
	// Path is where the file is.
	Path string
	// Text is what the editor shows: the disk's, plus what was typed.
	Text string
	// Dirty is whether Text has changes the disk does not have.
	Dirty bool
	// Version counts the changes to Text, so that work done for an older
	// text, such as a preview, is not applied to a newer one.
	Version uint64
	// Encoding is how the text became the bytes on disk, and becomes
	// those bytes again when it is saved.
	Encoding textfile.Encoding

	mtime time.Time
	size  int64
}

// Open reads the file at path into a note that has no changes yet. The
// byte order mark, when there is one, says how the bytes are read.
func Open(path string) (*Note, error) {
	return open(path, textfile.EncodingAuto)
}

// OpenEncoded reads the file with the encoding the caller classified,
// rather than asking the bytes again.
func OpenEncoded(path string, enc textfile.Encoding) (*Note, error) {
	return open(path, enc)
}

func open(path string, enc textfile.Encoding) (*Note, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", path, err)
	}
	if enc == textfile.EncodingAuto {
		enc = textfile.Detect(data)
	}
	text, err := textfile.Decode(data, enc)
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	n := &Note{Path: path, Text: text, Encoding: enc}
	n.remember(info)
	return n, nil
}

// OpenPreview reads the file at path as one the app only shows: what is
// known of it on the disk, and none of its text. A picture or a PDF is
// not read into an editor, which would hold a hundred megabytes of bytes
// no one can edit, and could not be written back as they were.
func OpenPreview(path string) (*Note, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", path, err)
	}
	n := &Note{Path: path}
	n.remember(info)
	return n, nil
}

// SetText is what the user typed. Text that did not change gives no new
// version, so nothing downstream is needlessly done again.
func (n *Note) SetText(text string) {
	if text == n.Text {
		return
	}
	n.Text = text
	n.Dirty = true
	n.Version++
}

// Save writes the text to the file: into a temporary file of the same
// directory first, which then replaces the file in one step. A crash
// leaves the old file, or the new one, never a mixture.
func (n *Note) Save() error {
	dir := filepath.Dir(n.Path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(n.Path)+".tmp")
	if err != nil {
		return fmt.Errorf("save %s: %w", n.Path, err)
	}
	name := tmp.Name()
	defer func() {
		// A save that failed leaves no temporary file behind.
		if err != nil {
			tmp.Close()
			os.Remove(name)
		}
	}()
	data, err := textfile.Encode(n.Text, n.Encoding)
	if err != nil {
		return fmt.Errorf("encode %s: %w", n.Path, err)
	}
	if _, err = tmp.Write(data); err != nil {
		return fmt.Errorf("save %s: %w", n.Path, err)
	}
	if err = tmp.Sync(); err != nil {
		return fmt.Errorf("save %s: %w", n.Path, err)
	}
	if err = tmp.Close(); err != nil {
		return fmt.Errorf("save %s: %w", n.Path, err)
	}
	// Windows will not rename over a file that is there.
	if err = os.Remove(n.Path); err != nil && !errors.Is(err,
		os.ErrNotExist) {
		return fmt.Errorf("save %s: %w", n.Path, err)
	}
	if err = os.Rename(name, n.Path); err != nil {
		return fmt.Errorf("save %s: %w", n.Path, err)
	}
	info, err := os.Stat(n.Path)
	if err != nil {
		return fmt.Errorf("save %s: %w", n.Path, err)
	}
	n.remember(info)
	n.Dirty = false
	return nil
}

// DiskChanged reports whether the file on disk is not the one this note
// read or wrote last: another program edited it, or the user edited it in
// another app.
func (n *Note) DiskChanged() bool {
	info, err := os.Stat(n.Path)
	if err != nil {
		return false
	}
	return !info.ModTime().Equal(n.mtime) || info.Size() != n.size
}

// TakeExternalText takes the text on disk when the file changed outside
// the app. It returns false when there is nothing to take: either the
// file did not change, or it changed into exactly what is here, in which
// case only what is remembered of it is brought up to date, so that the
// editor's caret is never moved for nothing.
func (n *Note) TakeExternalText() (string, bool) {
	if !n.DiskChanged() {
		return "", false
	}
	info, err := os.Stat(n.Path)
	if err != nil {
		return "", false
	}
	data, err := os.ReadFile(n.Path)
	if err != nil {
		return "", false
	}
	enc := textfile.Detect(data)
	text, err := textfile.Decode(data, enc)
	if err != nil {
		return "", false
	}
	if text == n.Text {
		n.remember(info)
		n.Encoding = enc
		return "", false
	}
	n.Text = text
	n.Dirty = false
	n.Version++
	n.Encoding = enc
	n.remember(info)
	return text, true
}

// remember keeps what the file on disk looked like, to tell later whether
// it changed.
func (n *Note) remember(info os.FileInfo) {
	n.mtime = info.ModTime()
	n.size = info.Size()
}
