// Package locale holds the messages of the app, one catalog to a
// language: en.json, the language the app is written in and the one the
// others fall back to, and zh-Hans.json. `mygo generate` writes
// catalog_gen.go from them, whose functions say a message in the
// language the app shows, and which registers the catalogs with MyGo, so
// that the app is written in the languages they hold.
//
// A message is named after what it says, and not after the words of it:
// menuOpenVault is "Open Vault…" in English and "打开笔记库…" in Chinese.
// The language is the configuration's, which the settings change; the app
// says it to MyGo (mygo.App.SetLocale), whose own controls, menus and
// dialogs follow it too.
//
// The catalogs are a package of their own, as MyGo wants them, so that
// every layer may say them: internal/app for the menus, the dialogs and
// the windows it opens, internal/view for what it draws.
package locale
