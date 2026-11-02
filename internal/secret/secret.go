// Package secret is a key kept where the system keeps keys, rather than
// in the settings that are written to disk.
//
// Where that is, is the system's to say: macOS keeps them in the
// Keychain. It is reached with the command the system ships rather than
// with the frameworks of the system, for the app is built without cgo
// and is to stay that way.
//
// A key is kept under a name made up for it, and not under the name of
// the model it is for: a model called by a new name keeps the key it had.
// A system with nowhere to keep a key says so of every request, and the
// key stays in the settings rather than being lost.
package secret

import "errors"

// Service is the name the keys are kept under: the name of the
// application, for whose keys they are.
const Service = "Xinkgo"

// ErrNoKeyring is what a system that keeps no keyring the app can reach
// says of every request.
var ErrNoKeyring = errors.New(
	"this system keeps no keyring the app can reach")
