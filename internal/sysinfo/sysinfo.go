// Package sysinfo describes the running app and the machine it runs on.
//
// The values are what an about window shows and what a web window asks Go
// for, and they are plain data, so the package has nothing to do with the
// app's lifetime and is testable on its own.
package sysinfo

import "runtime"

// Info is a snapshot of the app and its machine.
type Info struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	GoVersion string `json:"goVersion"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

// Current returns the information of the running app.
func Current(name, version string) Info {
	return Info{
		Name:      name,
		Version:   version,
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}
}

// String is one line describing the app, for titles and logs.
func (i Info) String() string {
	return i.Name + " " + i.Version + " (" + i.OS + "/" + i.Arch + ")"
}
