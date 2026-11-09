// Package media says what a file is by its name: whether the app can edit
// it as text, whether it is an image, which icon it gets, and which
// language the editor highlights it as.
//
// It is a table over extensions, with no I/O, so the file tree can decide
// what to show without touching the disk again.
package media

import (
	"path/filepath"
	"strings"

	"github.com/helightxu/xinkgo/internal/locale"
	"github.com/helightxu/xinkgo/internal/textfile"
)

// Kind is what a file is, which decides its icon and whether the app opens
// it in the editor.
type Kind int

// The kinds the file tree shows.
const (
	KindDir Kind = iota
	KindMarkdown
	KindImage
	KindPDF
	KindCode
	KindData
	KindArchive
	KindSpreadsheet
	KindAudio
	KindVideo
	KindOther
	KindText
)

// String is what the kind is called, as the file view heads a file with
// it.
func (k Kind) String() string {
	switch k {
	case KindDir:
		return locale.MediaKindDir()
	case KindMarkdown:
		return locale.MediaKindMarkdown()
	case KindImage:
		return locale.MediaKindImage()
	case KindPDF:
		return locale.MediaKindPDF()
	case KindCode:
		return locale.MediaKindCode()
	case KindData:
		return locale.MediaKindData()
	case KindArchive:
		return locale.MediaKindArchive()
	case KindSpreadsheet:
		return locale.MediaKindSpreadsheet()
	case KindAudio:
		return locale.MediaKindAudio()
	case KindVideo:
		return locale.MediaKindVideo()
	case KindText:
		return locale.MediaKindText()
	default:
		return locale.MediaKindOther()
	}
}

// AIDirName is the directory the AI writes its own files to. It is never
// shown in the tree, never indexed, and never offered to open.
const AIDirName = ".xink"

var (
	imageExts = map[string]bool{
		".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
		".webp": true, ".bmp": true, ".tiff": true, ".svg": true,
		".avif": true, ".heic": true, ".ico": true,
	}
	archiveExts = map[string]bool{
		".zip": true, ".tar": true, ".gz": true, ".bz2": true, ".xz": true,
		".7z": true, ".rar": true, ".tgz": true, ".zst": true,
	}
	spreadsheetExts = map[string]bool{
		".xls": true, ".xlsx": true, ".numbers": true,
	}
	audioExts = map[string]bool{
		".mp3": true, ".wav": true, ".ogg": true, ".flac": true,
		".m4a": true, ".aac": true, ".opus": true,
	}
	videoExts = map[string]bool{
		".mp4": true, ".mov": true, ".mkv": true, ".webm": true,
		".avi": true, ".m4v": true,
	}
	// dataExts are the files the editor formats and checks.
	dataExts = map[string]string{
		".json": "json", ".yaml": "yaml", ".yml": "yaml", ".toml": "toml",
	}
)

// KindOf says what the file at path is, by its name. A name that says
// nothing is left to the content of the file, which the tree records on
// the entry when it scans.
func KindOf(path string) Kind {
	name := filepath.Base(path)
	if name == AIDirName {
		return KindDir
	}
	ext := strings.ToLower(filepath.Ext(name))
	info, known := textfile.ByName(path)
	switch {
	case ext == ".md" || ext == ".markdown":
		return KindMarkdown
	case imageExts[ext]:
		return KindImage
	case ext == ".pdf":
		return KindPDF
	case dataExts[ext] != "":
		return KindData
	case known && info.Text && info.Language != "":
		return KindCode
	case known && info.Text:
		return KindText
	case archiveExts[ext]:
		return KindArchive
	case spreadsheetExts[ext]:
		return KindSpreadsheet
	case audioExts[ext]:
		return KindAudio
	case videoExts[ext]:
		return KindVideo
	default:
		return KindOther
	}
}

// IsImage reports whether the file is a picture the preview can show.
func IsImage(path string) bool {
	return imageExts[strings.ToLower(filepath.Ext(path))]
}

// IsPDF reports whether the file is one the system's own viewer shows:
// the app has no renderer of its own for them, so a PDF is previewed by
// the program the machine opens it with.
func IsPDF(path string) bool {
	return strings.ToLower(filepath.Ext(path)) == ".pdf"
}

// IsEditableText reports whether the name alone says the app opens the
// file in the editor. A name that says nothing is classified when the
// file is opened, or by the tree when it filters what it shows.
func IsEditableText(path string) bool {
	info, _ := textfile.ByName(path)
	return info.Text
}

// IsMarkdown reports whether the file is a note.
func IsMarkdown(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".md" || ext == ".markdown"
}

// Language is the language the editor highlights the file as, or "" when
// it is plain text. The names are MyGo's, so a fenced code block asks for
// the same tokenizer as the editor.
func Language(path string) string {
	return textfile.Language(path)
}

// IsAIDir reports whether name is the directory the AI keeps to itself,
// or something inside it: nothing of it is shown, indexed or opened.
func IsAIDir(name string) bool {
	for _, part := range strings.Split(filepath.Clean(name),
		string(filepath.Separator)) {
		if part == AIDirName {
			return true
		}
	}
	return false
}
