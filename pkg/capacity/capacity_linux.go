// Public Domain (-) 2018-present, The Elko Authors.
// See the Elko UNLICENSE file for details.

package capacity

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os/exec"
	"strconv"
)

var (
	buffers     float64
	cached      float64
	free        float64
	initialised bool
	total       float64
	pagesize    float64
	prevFree    float64
	prevPageout uint32
)

func cpuinfo() (float64, error) {
	out, err := ioutil.ReadFile("/proc/loadavg")
	if err != nil {
		return 0, err
	}
	split := bytes.Fields(out)
	if len(split) != 5 {
		return 0, fmt.Errorf(
			"unexpected read from /proc/loadavg: %s",
			string(out))
	}
	load, err := strconv.ParseFloat(split[0], 64)
	if err != nil {
		return 0, err
	}
	return load, nil
}

// TODO(tav): Possibly track the values of pgfree and pgmajfault.
func meminfo() (float64, error) {
	if !initialised {
		out, err := exec.Command("getconf", "PAGESIZE").Output()
		if err == nil {
			out = bytes.TrimSpace(out)
			pagesize, err = strconv.ParseFloat(string(out), 64) / 1024
		}
		if err != nil {
			pagesize = 4
		}
		initialised = true
	}
	out, err := ioutil.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, err
	}
	var split [][]byte
	for _, line := range bytes.Split(out, []byte{'\n'}) {
		split = bytes.Fields(line)
		if len(split) != 3 {
			return 0, fmt.Errorf(
				"unexpected read from /proc/meminfo: %s",
				string(line))
		}
		switch split[0] {
		case "MemTotal:":
			total, err = strconv.ParseFloat(split[1], 64)
			if err != nil {
				return 0, err
			}
		case "MemFree:":
			free, err = strconv.ParseFloat(split[1], 64)
			if err != nil {
				return 0, err
			}
		case "Buffers:":
			buffers, err = strconv.ParseFloat(split[1], 64)
			if err != nil {
				return 0, err
			}
		case "Cached:":
			cached, err = strconv.ParseFloat(split[1], 64)
			if err != nil {
				return 0, err
			}
		}
	}
	load := (total - free - buffers - cached) / total
	out, err = ioutil.ReadFile("/proc/vmstat")
	if err != nil {
		return 0, err
	}
	var pgpgout uint32
	next := false
	for i, elem := range bytes.Fields(out) {
		if next {
			pgpgout, err = uint32(strconv.ParseUint(elem, 10, 32))
			if err != nil {
				return 0, err
			}
			break
		}
		if elem == "pgpgout" {
			next = true
		}
	}
	if !next {
		return 0, fmt.Errorf(
			"could not find a value for pgpgout from reading /proc/vmstat")
	}
	if prevFree < 1 {
		load += float64(pgpgout-prevPageout) / 1
	} else {
		load += float64(pgpgout-prevPageout) / prevFree
	}
	prevFree = free
	prevPageout = pgpgout
	return load, nil
}
