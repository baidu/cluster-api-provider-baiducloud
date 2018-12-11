package utils

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/golang/glog"
)

// RemoteSSHBashScript executes commnad on remote machine
func RemoteSSHBashScript(user, ip, password, cmd string) (string, error) {
	// TODO use public key
	c := exec.Command("sshpass", "-p", password, "ssh", "-o", "StrictHostKeyChecking=no", "-o", "UserKnownHostsFile=/dev/null", "-q", user+"@"+ip, "bash", "-c", cmd)
	glog.Infof("cmd args %s", c.Args)
	out, err := c.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error: %v, output: %s", err, string(out))
	}
	result := strings.TrimSpace(string(out))
	return result, nil
}

func RemoteSSHCommand(user, ip, password, cmd string) (string, error) {
	// TODO use public key
	c := exec.Command("sshpass", "-p", password, "ssh", "-o", "StrictHostKeyChecking=no", "-o", "UserKnownHostsFile=/dev/null", "-q", user+"@"+ip, cmd)
	glog.Infof("cmd args %s", c.Args)
	out, err := c.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error: %v, output: %s", err, string(out))
	}
	result := strings.TrimSpace(string(out))
	return result, nil
}
