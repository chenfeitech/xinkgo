package plugin

import "strings"

// The values a first start uses for the built-in core plugins' options.
const (
	DefaultSearchEngine = "https://duckduckgo.com/?q={query}"
)

// CoreOptions are the choices of the three built-in core plugins. They are
// kept in plugins.json beside the switches, so one file says what the user
// turned on and how it is set up. A field the file leaves out takes its
// default; an explicit false is a choice and is kept.
type CoreOptions struct {
	DailyNotes DailyNotesOptions `json:"daily_notes"`
	Templates  TemplatesOptions  `json:"templates"`
	WebViewer  WebViewerOptions  `json:"web_viewer"`
}

// DailyNotesOptions is how a daily note is named and filled in.
type DailyNotesOptions struct {
	DateFormat    string `json:"date_format"`
	Folder        string `json:"folder"`
	TemplatePath  string `json:"template_path"`
	OpenOnStartup bool   `json:"open_on_startup"`
}

// TemplatesOptions is where templates live and how dates inside them look.
type TemplatesOptions struct {
	Folder     string `json:"folder"`
	DateFormat string `json:"date_format"`
	TimeFormat string `json:"time_format"`
}

// WebViewerOptions is the browser window's own setup.
type WebViewerOptions struct {
	Homepage     string `json:"homepage"`
	SearchEngine string `json:"search_engine"`
	SaveFolder   string `json:"save_folder"`
	SaveTemplate string `json:"save_template"`
	AdBlock      bool   `json:"ad_block"`
	RuleFile     string `json:"rule_file"`
	AllowLocal   bool   `json:"allow_local"`
	ClearOnQuit  bool   `json:"clear_on_quit"`
}

// DefaultCoreOptions is the setup of a first start.
func DefaultCoreOptions() CoreOptions {
	return CoreOptions{
		DailyNotes: DailyNotesOptions{DateFormat: "YYYY-MM-DD"},
		Templates: TemplatesOptions{
			DateFormat: "YYYY-MM-DD", TimeFormat: "HH:mm",
		},
		WebViewer: WebViewerOptions{
			SearchEngine: DefaultSearchEngine,
			AdBlock:      true,
		},
	}
}

// Sanitized fills fields the file left out with their defaults and tidies
// paths. Explicit false values (OpenOnStartup, AdBlock, AllowLocal,
// ClearOnQuit) are left alone. Format validity is checked where the format
// is used, not here: this package stays pure data.
func (o CoreOptions) Sanitized() CoreOptions {
	d := DefaultCoreOptions()
	if strings.TrimSpace(o.DailyNotes.DateFormat) == "" {
		o.DailyNotes.DateFormat = d.DailyNotes.DateFormat
	}
	o.DailyNotes.Folder = cleanPath(o.DailyNotes.Folder)
	o.DailyNotes.TemplatePath = cleanPath(o.DailyNotes.TemplatePath)
	if strings.TrimSpace(o.Templates.DateFormat) == "" {
		o.Templates.DateFormat = d.Templates.DateFormat
	}
	if strings.TrimSpace(o.Templates.TimeFormat) == "" {
		o.Templates.TimeFormat = d.Templates.TimeFormat
	}
	o.Templates.Folder = cleanPath(o.Templates.Folder)
	if strings.TrimSpace(o.WebViewer.SearchEngine) == "" {
		o.WebViewer.SearchEngine = d.WebViewer.SearchEngine
	}
	o.WebViewer.Homepage = strings.TrimSpace(o.WebViewer.Homepage)
	o.WebViewer.SaveFolder = cleanPath(o.WebViewer.SaveFolder)
	o.WebViewer.SaveTemplate = cleanPath(o.WebViewer.SaveTemplate)
	o.WebViewer.RuleFile = cleanPath(o.WebViewer.RuleFile)
	return o
}

// cleanPath keeps a vault-relative path a path, not a Windows separator or
// a stray whitespace around it. Resolving it inside the vault is the app's
// work, not this package's.
func cleanPath(p string) string {
	return strings.TrimSpace(strings.ReplaceAll(p, "\\", "/"))
}
