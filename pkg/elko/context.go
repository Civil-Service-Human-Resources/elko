// Public Domain (-) 2018-present, The Elko Authors.
// See the Elko UNLICENSE file for details.

package elko

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
	"time"

	"github.com/tav/elko/pkg/protocol"
)

type Context struct {
	ID     string
	Header *protocol.Header
}

func (c *Context) Call(svc string, args ...interface{}) *Call {
	return &Call{
		args: args,
		ctx:  c,
		svc:  svc,
	}
}

func (c *Context) Fire(svc string, args ...interface{}) error {
	err := WriteReq(c, 0, svc, args)
	if err != nil {
		c.ErrorData(fmt.Sprintf("Failed to encode request to service '%s'. Error: '%s'", svc, err.Error()), args)
		return toError(err)
	}
	return nil
}

func (c *Context) Log(args ...interface{}) {
	c.Fire("log.persist", &protocol.LogEntry{
		DeployID:   DeployID,
		InstanceID: Instance,
		ServiceID:  Service,
		Context:    c.ID,
		Message:    fmt.Sprint(args...),
		Timestamp:  time.Now().UTC(),
	})
}

func (c *Context) Logf(format string, args ...interface{}) {
	c.Fire("log.persist", &protocol.LogEntry{
		DeployID:   DeployID,
		InstanceID: Instance,
		ServiceID:  Service,
		Context:    c.ID,
		Message:    fmt.Sprintf(format, args...),
		Timestamp:  time.Now().UTC(),
	})
}

func (c *Context) LogData(message string, data interface{}) {
	c.Fire("log.persist", &protocol.LogEntry{
		DeployID:   DeployID,
		InstanceID: Instance,
		ServiceID:  Service,
		Context:    c.ID,
		Data:       data,
		Message:    message,
		Timestamp:  time.Now().UTC(),
	})
}

func (c *Context) Error(args ...interface{}) {
	e := protocol.NewLogEntry(2)
	e.DeployID = DeployID
	e.InstanceID = Instance
	e.ServiceID = Service
	e.Context = c.ID
	e.Error = true
	e.Message = fmt.Sprint(args...)
	e.Timestamp = time.Now().UTC()
	c.Fire("log.persist", e)
}

func (c *Context) Errorf(format string, args ...interface{}) {
	e := protocol.NewLogEntry(2)
	e.DeployID = DeployID
	e.InstanceID = Instance
	e.ServiceID = Service
	e.Context = c.ID
	e.Error = true
	e.Message = fmt.Sprintf(format, args...)
	e.Timestamp = time.Now().UTC()
	c.Fire("log.persist", e)
}

func (c *Context) ErrorData(message string, data interface{}) {
	e := protocol.NewLogEntry(2)
	e.DeployID = DeployID
	e.InstanceID = Instance
	e.ServiceID = Service
	e.Context = c.ID
	e.Data = data
	e.Error = true
	e.Message = message
	e.Timestamp = time.Now().UTC()
	c.Fire("log.persist", e)
}

func NewContext() *Context {
	contextBuf := &bytes.Buffer{}
	contextBuf.Write(serviceCtx)
	muCtx.Lock()
	lastCtxID += 1
	nextID := lastCtxID
	muCtx.Unlock()
	ctxBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(ctxBuf, nextID)
	contextBuf.Write(ctxBuf)
	contextBuf.WriteString(Service)
	return &Context{
		Header: &protocol.Header{},
		ID:     string(contextBuf.Bytes()[:]), //should ID just be a byte slice?
	}
}

type WebResponse struct {
	Status int               `codec:"status"`
	Header map[string]string `codec:"header"`
	Body   []byte            `codec:"body"`
}

type WebRequest struct {
	Header    map[string]string   `codec:"header"`
	Host      string              `codec:"host"`
	Path      string              `codec:"path"`
	Method    string              `codec:"method"`
	PathArgs  []string            `codec:"pathArgs"`
	QueryArgs map[string]string   `codec:"queryArgs"`
	Cookies   map[string][]string `codec:"cookies"`
	Scheme    string              `codec:"scheme"`
}

type WebContext struct {
	*Context
	Request *WebRequest
	status  int
	header  map[string]string
	cookies map[string][]string
}

func (c *WebContext) CacheResponse(d time.Duration) {
	c.SetHeader("cache-control", fmt.Sprintf("public, max-age=%d", d/time.Second))
}

func (c *WebContext) SetHeader(name string, value string) {
	if c.header == nil {
		c.header = map[string]string{}
	}
	c.header[strings.ToLower(name)] = value
}

func (c *WebContext) SetStatus(code int) {
	c.status = code
}

func (c *WebContext) Redirect(url string, permanent bool) {
	if permanent {
		c.status = 301
	} else {
		c.status = 302
	}
	c.SetHeader("location", url)
}

func toError() {

}
