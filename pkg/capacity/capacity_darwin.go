// Public Domain (-) 2018-present, The Elko Authors.
// See the Elko UNLICENSE file for details.

package capacity

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
)

func cpuinfo() (float64, error) {
	out, err := sysctl("vm.loadavg")
	if err != nil {
		return 0, err
	}
	split := bytes.Fields(out)
	if len(split) != 5 {
		return 0, fmt.Errorf(
			"unexpected read from sysctl -n vm.loadavg: %s",
			string(out))
	}
	load, err := strconv.ParseFloat(string(split[1]), 64)
	if err != nil {
		return 0, err
	}
	return load, nil
}

func meminfo() (float64, error) {
	out, err := sysctl("vm.vm_page_free_target", "vm.page_free_count")
	if err != nil {
		return 0, err
	}
	split := bytes.Fields(out)
	if len(split) != 2 {
		return 0, fmt.Errorf(
			"unexpected read from sysctl -n vm.vm_page_free_target vm.page_free_count: %s",
			string(out))
	}
	tgt, err := strconv.ParseFloat(string(split[0]), 64)
	if err != nil {
		return 0, err
	}
	free, err := strconv.ParseFloat(string(split[1]), 64)
	if err != nil {
		return 0, err
	}
	if free < 1 {
		free = 1
	}
	return tgt / free, nil
}

func sysctl(args ...string) ([]byte, error) {
	buf := &bytes.Buffer{}
	cmd := exec.Cmd{
		Path:   "/usr/sbin/sysctl",
		Args:   append([]string{"sysctl", "-n"}, args...),
		Stdout: buf,
	}
	err := cmd.Run()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
