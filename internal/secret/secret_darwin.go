//go:build darwin

package secret

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// security is the command macOS ships for the Keychain.
const security = "/usr/bin/security"

// howLong is how long the system is given to answer: it is asked away
// from the window, but it is not waited for for ever.
const howLong = 10 * time.Second

// found is whether the keyring can be reached, found out once and no
// more: the window asks anew every frame, and the answer does not change
// while the app is running.
var found = sync.OnceValue(func() bool {
	_, err := exec.LookPath(security)
	return err == nil
})

// Available is whether this system keeps a keyring the app can reach.
func Available() bool { return found() }

// Get is the key kept under that name. A key that is not there comes back
// as an error, as does one the system would not give up.
func Get(name string) (string, error) {
	if !Available() {
		return "", ErrNoKeyring
	}
	if name == "" {
		return "", errors.New("no name was given")
	}
	out, err := ask("find-generic-password", "-a", name, "-s", Service,
		"-w")
	if err != nil {
		return "", err
	}
	return string(bytes.TrimRight(out, "\n")), nil
}

// Put keeps key under that name, taking out what was kept under it
// before.
//
// The key is written to the command rather than spelled out beside it: a
// key spelled out beside it is one any other process may read out of the
// list of what is running. What the command wants is not agreed on — it
// reads the key, or it wants it spelled out beside it, depending on the
// version — so both are offered, and whichever it took is known by
// asking for the key back.
func Put(name, key string) error {
	if !Available() {
		return ErrNoKeyring
	}
	if name == "" {
		return errors.New("no name was given")
	}
	if key == "" {
		return errors.New("no key was given")
	}
	// What is written is the key and nothing else: a newline written
	// with the key is kept as part of it.
	cmd := command("add-generic-password", "-a", name, "-s", Service, "-U")
	cmd.Stdin = strings.NewReader(key)
	if _, err := cmd.Output(); err == nil {
		if err := kept(name, key); err == nil {
			return nil
		}
	}
	if _, err := ask("add-generic-password", "-a", name, "-s", Service,
		"-U", "-w", key); err != nil {
		return err
	}
	return kept(name, key)
}

// kept is whether the keyring gives that key back.
//
// A key the keyring will not give back is not one that has been kept,
// and saying that it has been is how a key is lost: it is said to have
// failed instead, so that whatever held it before goes on holding it.
func kept(name, key string) error {
	got, err := Get(name)
	if err != nil {
		return err
	}
	if got != key {
		// What came back is not said, for it is a key: how much of
		// it there is says enough to know which way it went wrong.
		return fmt.Errorf("the keyring gave back %d letters for %d",
			len(got), len(key))
	}
	return nil
}

// Drop takes the key kept under that name out of the keyring.
func Drop(name string) error {
	if !Available() {
		return ErrNoKeyring
	}
	if name == "" {
		return nil
	}
	_, err := ask("delete-generic-password", "-a", name, "-s", Service)
	return err
}

// ask is the system asked, and what it said back.
func ask(args ...string) ([]byte, error) {
	return command(args...).Output()
}

// command is the system asked, and given howLong to answer.
func command(args ...string) *exec.Cmd {
	ctx, cancel := context.WithTimeout(context.Background(), howLong)
	cmd := exec.CommandContext(ctx, security, args...)
	// The waiting is over when the answer has come: the command is not
	// left running behind it.
	cmd.Cancel = func() error {
		cancel()
		return cmd.Process.Kill()
	}
	return cmd
}
