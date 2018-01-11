// Public Domain (-) 2018-present, The Elko Authors.
// See the Elko UNLICENSE file for details.

package consul

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/tav/golly/log"
)

const kv = "/v1/kv/"

var NotFound = errors.New("consul: key not found")

var (
	client    = &http.Client{}
	endpoint  = ""
	endpoints = []string{}
	i         = 0
	root      = ""
)

type Item struct {
	Key         string
	ModifyIndex uint64
	Value       []byte
	// CreateIndex uint64
	// Flags       uint64
	// LockIndex   uint64
	// Session     string
}

type Listing struct {
	Index uint64
	Items []*Item
}

func check(resp *http.Response) error {
	if resp.StatusCode != 200 {
		if resp.StatusCode == 404 {
			return NotFound
		}
		return fmt.Errorf("consul: got a non-200 response code: %d", resp.StatusCode)
	}
	return nil
}

func delete(path *url.URL) error {
	path.Host = getEndpoint()
	path.Scheme = "http"
	resp, err := client.Do(&http.Request{
		Method: "DELETE",
		URL:    path,
	})
	if err != nil {
		return err
	}
	resp.Body.Close()
	return check(resp)
}

func get(path *url.URL) ([]*Item, string, error) {
	path.Host = getEndpoint()
	path.Scheme = "http"
	resp, err := client.Do(&http.Request{
		Method: "GET",
		URL:    path,
	})
	if err != nil {
		return nil, "", err
	}
	err = check(resp)
	if err != nil {
		resp.Body.Close()
		return nil, "", err
	}
	items := []*Item{}
	body := json.NewDecoder(resp.Body)
	err = body.Decode(&items)
	resp.Body.Close()
	if err != nil {
		return nil, "", err
	}
	return items, resp.Header.Get("X-Consul-Index"), err
}

func getEndpoint() string {
	if endpoint != "" {
		return endpoint
	}
	if len(endpoints) > 0 {
		i += 1
		return endpoints[i%len(endpoints)]
	}
	log.Fatal("consul: endpoint not set")
	return ""
}

func getItem(path *url.URL) (*Item, error) {
	items, _, err := get(path)
	if err != nil {
		return nil, err
	}
	return items[0], nil
}

func getListing(path *url.URL) (*Listing, error) {
	items, index, err := get(path)
	if err != nil {
		return nil, err
	}
	list := &Listing{
		Items: items,
	}
	list.Index, err = strconv.ParseUint(index, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("consul: invalid X-Consul-Index value: %s", err)
	}
	return list, err
}

func put(value []byte, path *url.URL) (bool, error) {
	buf := bytes.NewBuffer(value)
	path.Host = getEndpoint()
	path.Scheme = "http"
	resp, err := client.Do(&http.Request{
		Body:          ioutil.NopCloser(buf),
		ContentLength: int64(buf.Len()),
		Method:        "PUT",
		URL:           path,
	})
	if err != nil {
		return false, err
	}
	err = check(resp)
	if err != nil {
		resp.Body.Close()
		return false, err
	}
	ok := false
	body := json.NewDecoder(resp.Body)
	err = body.Decode(&ok)
	resp.Body.Close()
	if err != nil {
		return false, err
	}
	return ok, nil
}

func urlpath(key string) string {
	return kv + root + strings.TrimPrefix(key, "/")
}

func CAS(key string, value []byte, index uint64) (bool, error) {
	return put(value, &url.URL{
		Path:     urlpath(key),
		RawQuery: "cas=" + strconv.FormatUint(index, 10),
	})
}

func Delete(key string) error {
	return delete(&url.URL{
		Path: urlpath(key),
	})
}

func DeletePrefix(prefix string) error {
	return delete(&url.URL{
		Path:     urlpath(prefix),
		RawQuery: "recurse",
	})
}

func Get(key string) (*Item, error) {
	return getItem(&url.URL{
		Path:     urlpath(key),
		RawQuery: "consistent",
	})
}

func GetMulti(prefix string) (*Listing, error) {
	return getListing(&url.URL{
		Path:     urlpath(prefix),
		RawQuery: "recurse&consistent",
	})
}

func List(prefix string, separator string) ([]string, error) {
	query := "keys&consistent"
	if separator != "" {
		query += "&separator=" + url.QueryEscape(separator)
	}
	resp, err := client.Do(&http.Request{
		Method: "GET",
		URL: &url.URL{
			Host:     getEndpoint(),
			Path:     urlpath(prefix),
			RawQuery: query,
			Scheme:   "http",
		},
	})
	if err != nil {
		return nil, err
	}
	err = check(resp)
	if err != nil {
		resp.Body.Close()
		return nil, err
	}
	keys := []string{}
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&keys)
	resp.Body.Close()
	return keys, err
}

func Put(key string, value []byte) (bool, error) {
	return put(value, &url.URL{
		Path: urlpath(key),
	})
}

func SetEndpoint(servers ...string) {
	if len(servers) == 1 {
		endpoint = servers[0]
	} else {
		endpoints = servers
	}
}

func SetRootPrefix(prefix string) {
	root = prefix
}

func Wait(key string, interval time.Duration, index uint64) (*Item, error) {
	return getItem(&url.URL{
		Path:     urlpath(key),
		RawQuery: "consistent&wait=" + interval.String() + "&index=" + strconv.FormatUint(index, 10),
	})
}

func WaitMulti(prefix string, interval time.Duration, index uint64) (*Listing, error) {
	return getListing(&url.URL{
		Path:     urlpath(prefix),
		RawQuery: "recurse&consistent&wait=" + interval.String() + "&index=" + strconv.FormatUint(index, 10),
	})
}
