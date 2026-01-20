package app

import "os/exec"

var LookPath = exec.LookPath

func HasExecutable(name string) bool {
	if name == "" {
		return false
	}
	_, err := LookPath(name)
	return err == nil
}
