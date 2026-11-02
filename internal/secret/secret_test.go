package secret

import "testing"

// TestGet_aKeyOfNoNameIsNoKeyAtAll checks that the system is not asked
// about a name that was never made: a key kept under nothing is one no
// model can be asked with.
func TestGet_aKeyOfNoNameIsNoKeyAtAll(t *testing.T) {
	if _, err := Get(""); err == nil {
		t.Fatal("a key came back for no name")
	}
}

// TestPut_aKeyThatIsNothingIsNotKept checks that keeping nothing is said
// to have failed: a model whose key could not be kept is one the
// settings must leave the key of, and silence would say otherwise.
func TestPut_aKeyThatIsNothingIsNotKept(t *testing.T) {
	if err := Put("name", ""); err == nil {
		t.Fatal("nothing was said of a key that is nothing")
	}
	if err := Put("", "sk-one"); err == nil {
		t.Fatal("nothing was said of a name that is nothing")
	}
}

// TestPut_whenThereIsNoKeyringTheKeyIsNotLost checks that a system with
// nowhere to keep a key says so of every request, rather than taking the
// key and losing it: the key stays in the settings instead.
func TestPut_whenThereIsNoKeyringTheKeyIsNotLost(t *testing.T) {
	if Available() {
		t.Skip("this system keeps a keyring the app can reach")
	}
	if err := Put("name", "sk-one"); err != ErrNoKeyring {
		t.Fatalf("err = %v, want it said that there is no keyring", err)
	}
	if _, err := Get("name"); err != ErrNoKeyring {
		t.Fatalf("err = %v, want it said that there is no keyring", err)
	}
	if err := Drop("name"); err != ErrNoKeyring {
		t.Fatalf("err = %v, want it said that there is no keyring", err)
	}
}
