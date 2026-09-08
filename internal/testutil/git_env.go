package testutil

import "os"

// IsolateGitEnvironment prevents host-wide signing and editor preferences from
// changing disposable-repository test behavior.
func IsolateGitEnvironment() {
	for _, setting := range [][2]string{
		{"GIT_CONFIG_GLOBAL", os.DevNull},
		{"GIT_CONFIG_SYSTEM", os.DevNull},
		{"GIT_EDITOR", "true"},
		{"GIT_SEQUENCE_EDITOR", "true"},
	} {
		if err := os.Setenv(setting[0], setting[1]); err != nil {
			panic(err)
		}
	}
}
