// Public Domain (-) 2018-present, The Elko Authors.
// See the Elko UNLICENSE file for details.

package lease

import (
	"fmt"
	"time"

	"github.com/tav/elko/pkg/consul"
)

type Entry struct {
	cleanup func(string) error
	key     string
	period  time.Duration
}

// Acquire tries to secure a lease and returns the immediate expiry time.
func (e *Entry) Acquire() (time.Time, error) {
	expiry := time.Now().UTC().Add(e.period)
	ok, err := consul.CAS(e.key, []byte{'A'}, 0)
	if err != nil {
		return expiry, err
	}
	if !ok {
		return expiry, fmt.Errorf(
			"lease: the lease for key %s already exists", e.key)
	}
	return expiry, nil
}

// Contest tries to determine the state of the lease holder and returns a
// boolean indicating if it is still alive. If the lease is currently acquired,
// it will contest it and wait for the lease holder to assert itself within the
// lease duration. If the lease holder doesn't assert itself or has already been
// contested/invalidated, it will initiate a clean up and remove the lease.
func (e *Entry) Contest() (bool, error) {
	k := e.key
	i, err := consul.Get(k)
	if err != nil {
		if err == consul.NotFound {
			return false, nil
		}
		return false, err
	}
	if i.Value[0] == 'A' {
		// In between, it is possible for something else to have:
		//
		// - Invalidated the lease
		// - Contested the lease
		// - Contested the lease and had it re-acquired (potentially multiple times)
		// - Removed the lease after contesting/invalidating it
		_, err = consul.CAS(k, []byte{'C'}, i.ModifyIndex)
		if err != nil {
			// TODO(tav): find out if CAS can return a NotFound error.
			return false, err
		}
	}
	i, err = consul.Get(k)
	if err != nil {
		if err == consul.NotFound {
			return false, nil
		}
		return false, err
	}
	if i.Value[0] == 'A' {
		return true, nil
	}
	invalidated := i.Value[0] == 'I'
	j, err := consul.Wait(k, e.period, i.ModifyIndex)
	if err != nil {
		if err == consul.NotFound {
			return false, nil
		}
		return false, err
	}
	if j.Value[0] == 'I' {
		if !invalidated {
			time.Sleep(e.period)
		}
		return false, e.Cleanup()
	}
	if i.ModifyIndex == j.ModifyIndex {
		return false, e.Cleanup()
	}
	return true, nil
}

// Cleanup runs the registered cleanup function and removes the lease.
func (e *Entry) Cleanup() error {
	if e.cleanup != nil {
		err := e.cleanup(e.key)
		if err != nil {
			return err
		}
	}
	return consul.Delete(e.key)
}

// Invalidate informs the lease holder that the lease has been forcibly revoked.
// Once Invalidate has been called, all other processes need to wait a minimum
// of the lease duration before they can safely proceed as if the lease was no
// longer valid. Invalidate also initiates cleanup and removes the lease.
func (e *Entry) Invalidate() error {
	i, err := consul.Get(e.key)
	if err != nil {
		if err == consul.NotFound {
			return nil
		}
		return err
	}
	if i.Value[0] == 'I' {
		time.Sleep(e.period)
		return e.Cleanup()
	}
	ok, err := consul.CAS(e.key, []byte{'I'}, i.ModifyIndex)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf(
			"lease: failed to invalidate the lease for key %s", e.key)
	}
	time.Sleep(e.period)
	return e.Cleanup()
}

// Maintain tries to hold onto a given lease for as long as possible. If another
// process contests the lease, then as long as there is still time left in the
// lease duration, then it will automatically try and assert itself. However, if
// another process has forcibly revoked the lease with an Invalidate call, then
// the lease is no longer maintained.
func (e *Entry) Maintain(expiry time.Time) error {
	var (
		err error
		i   *consul.Item
		now time.Time
		ok  bool
	)
	attempts := 0
	hold := []byte{'A'}
	k := e.key
	period := e.period
	for {
		time.Sleep(period / 7)
		now = time.Now().UTC()
		if now.After(expiry) {
			return fmt.Errorf(
				"lease: the lease for key %s has been invalidated", e.key)
		}
		i, err = consul.Get(k)
		if err != nil {
			if err == consul.NotFound {
				return fmt.Errorf(
					"lease: the lease for key %s has been invalidated", e.key)
			}
			attempts++
			if attempts == 6 {
				return err
			}
			continue
		}
		if i.Value[0] == 'I' {
			return fmt.Errorf(
				"lease: the lease for key %s has been invalidated", e.key)
		}
		if i.Value[0] == 'A' {
			attempts = 0
			expiry = time.Now().UTC().Add(period)
		} else {
			now = time.Now().UTC()
			if now.After(expiry) {
				return fmt.Errorf(
					"lease: the lease for key %s has been invalidated", e.key)
			}
			ok, err = consul.CAS(k, hold, i.ModifyIndex)
			if err != nil {
				attempts++
				if attempts == 6 {
					return err
				}
			} else {
				if ok {
					attempts = 0
					expiry = time.Now().UTC().Add(period)
				} else {
					return fmt.Errorf(
						"lease: the lease for key %s has been invalidated", e.key)
				}
			}
		}
	}
}

// New returns a new lease entry for the specific key.
func New(key string, cleanup func(string) error, period time.Duration) *Entry {
	return &Entry{
		cleanup: cleanup,
		key:     "lease/" + key,
		period:  period,
	}
}
