//go:build !darwin

package secret

// Available is whether this system keeps a keyring the app can reach.
// Only macOS is reached for now: a key on another system stays in the
// settings rather than being handed to something that cannot be tried
// here.
func Available() bool { return false }

// Get is the key kept under that name: none, for there is nowhere it is
// kept.
func Get(name string) (string, error) { return "", ErrNoKeyring }

// Put keeps key under that name: it cannot, and says so, so that the key
// is left where it was.
func Put(name, key string) error { return ErrNoKeyring }

// Drop takes the key kept under that name out: there is nothing to take
// out.
func Drop(name string) error { return ErrNoKeyring }
