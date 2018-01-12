// Public Domain (-) 2018-present, The Elko Authors.
// See the Elko UNLICENSE file for details.

package freeport

import (
	"net"
)

func Get() (int, error) {
	l, err := net.ListenTCP("tcp", nil)
	if err != nil {
		return 0, err
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port, nil
}
