// Public Domain (-) 2018-present, The Elko Authors.
// See the Elko UNLICENSE file for details.

package protocol

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"runtime"
	"sync"
	"time"
)

func Decode(data []byte, v interface{}) error {
}

func Encode(w io.Writer, v interface{}) error {
}

// Data Structures
type LogEntry struct {
	Context    string      `protobuf:"ctx"                   json:"ctx"`
	Data       interface{} `protobuf:"data"                  json:"data"`
	DeployID   uint64      `protobuf:"deploy_id,omitempty"   json:"deploy_id,omitempty"`
	Error      bool        `protobuf:"error"                 json:"error"`
	File       string      `protobuf:"file,omitempty"        json:"file,omitempty"`
	InstanceID uint64      `protobuf:"instance_id,omitempty" json:"instance_id,omitempty"`
	Line       int         `protobuf:"line,omitempty"        json:"line,omitempty"`
	Message    string      `protobuf:"msg"                   json:"msg"`
	ServiceID  string      `protobuf:"service_id,omitempty"   json:"service_id,omitempty"`
	Stacktrace string      `protobuf:"stacktrace,omitempty"  json:"stacktrace,omitempty"`
	Timestamp  time.Time   `protobuf:"timestamp"             json:"timestamp"`
}

func NewLogEntry(depth int) *LogEntry {
	e := &LogEntry{}
	_, e.File, e.Line, _ = runtime.Caller(depth)
	buf := make([]byte, 4096)
	e.Stacktrace = string(buf[:runtime.Stack(buf, false)])
	return e
}

type ServiceHello struct {
	InstanceID uint64 `protobuf:"instanceID"`
}

type Reload struct {
	DeployID uint64 `protobuf:"deployID"`
}

type NodeHello struct {
	NodeID        string
	ForeignNodeID string
}

type Call struct {
	Context string `protobuf:"ctx"`
	Service string `protobuf:"service"`
}

type Request struct {
	Context    string   `protobuf:"ctx"`
	Service    string   `protobuf:"service"`
	Args       [][]byte `protobuf:"args"`
	Headers    Header   `protobuf:"headers"`
	ID         uint64   `protobuf:"id"`
	NodeID     string   `protobuf:"nodeID"`
	InstanceID uint64   `protobuf:"instanceID"`
}

type Header struct {
	Auth    string `protobuf:"auth"`
	TraceID string `protobuf:"traceID"`
}

type Response struct {
	Context    string `protobuf:"ctx"`
	NodeID     string `protobuf:"nodeID"`
	InstanceID uint64 `protobuf:"instanceID"`
}

type ResponseBody struct {
	Context    string `protobuf:"ctx"`
	NodeID     string `protobuf:"nodeID"`
	InstanceID uint64 `protobuf:"instanceID"`
	ID         uint64 `protobuf:"id"`
	Result     []byte `protobuf:result`
	Error      *Error `protobuf:error`
}

type Error struct {
	Type string        `protobuf:type`
	Args []interface{} `protobuf:args`
	Msg  string        `protobuf:message`
}

func (e Error) Error() string {
	if e.Type != "" {
		if e.Msg != "" {
			return fmt.Sprintf("%s: %s", e.Type, e.Msg)
		}
		if e.Args != nil {
			out, _ := json.Marshal(e.Args)
			return fmt.Sprintf("%s: %s", e.Type, out)
		}
	}
	if e.Msg != "" {
		return e.Msg // TODO(tav): decode args OR store as a list of values?
	}
	return "Error"
}

type Log struct {
	Context string      `protobuf:"ctx"`
	Type    string      `protobuf:"type"`
	Message string      `protobuf:"message"`
	Data    interface{} `protobuf:"data"`
}

var ErrConnectionClosed = errors.New("elko.protocol: connection closed")

var payloadPool = &sync.Pool{}

type Payload struct {
	Opcode byte
	Data   interface{}
	Body   []byte
}

type Conn struct {
	closed bool
	conn   net.Conn
	in     chan *Payload
	mu     sync.Mutex
	queue  chan *Payload
}

func (c *Conn) Write(opcode byte, data interface{}, body []byte) error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return ErrConnectionClosed
	}
	c.mu.Unlock()
	item := payloadPool.Get()
	c.mu.Lock()
	if c.in == nil {
		c.in = make(chan *Payload, 100)
		c.queue = make(chan *Payload)
	}
	c.mu.Unlock()
	if item == nil {
		c.in <- &Payload{opcode, data, body}
	} else {
		p := item.(*Payload)
		p.Opcode = opcode
		p.Data = data
		p.Body = body
		c.in <- p
	}
	return nil
}

func (c *Conn) Close() {
	c.mu.Lock()
	//close(c.in)
	//close(c.queue)
	c.closed = true
	c.conn.Close()
	c.mu.Unlock()
}

func (c *Conn) Run(h Handler, timeout time.Duration) {
	go c.StartReadLoop(h, timeout)
	go c.StartWriteLoop(timeout)
}

func (c *Conn) StartReadLoop(h Handler, timeout time.Duration) {
	var opcode byte
	dataBuf := make([]byte, 4096)
	var dataLen uint32
	lenBuf := make([]byte, 4)
	var err error
	data := new(bytes.Buffer)
	var n int
	var readBytes uint32
	conn := c.conn
	r := bufio.NewReader(conn)
	body := &bytes.Buffer{}
	for {
		if timeout != time.Duration(0) {
			conn.SetReadDeadline(time.Now().Add(timeout))
		}
		opcode, err = r.ReadByte()
		if err != nil {
			// reconnect?
			fmt.Printf("error: %v", err)
			conn.Close()
			break
			//c.cleanUp() -> tell the server to kill this connection -> runtime should try and reconnect
			// log if not eof client closes connection
		}
		if opcode >= 64 {
			readBytes = 0
			for readBytes < 4 {
				n, err = r.Read(lenBuf[readBytes:])
				if err != nil {
					fmt.Printf("error: %v", err)
					break
					//c.cleanUp()
					// log if not eof client closes connection
					return
				}
				readBytes += uint32(n)
			}
			dataLen = binary.BigEndian.Uint32(lenBuf)
			for dataLen > 0 {
				if dataLen < 4096 {
					n, err = r.Read(dataBuf[:dataLen])
				} else {
					n, err = r.Read(dataBuf)
				}
				if err != nil {
					fmt.Printf("error: %v", err)
					break
					//c.cleanUp()
					// log if not eof client closes connection
				}
				data.Write(dataBuf[:n])
				dataLen -= uint32(n)
			}
		}
		if opcode >= 128 {
			readBytes = 0
			for readBytes < 4 {
				n, err = r.Read(lenBuf[readBytes:])
				if err != nil {
					fmt.Printf("error: %v", err)
					break
					//c.cleanUp()
					// log if not eof client closes connection
				}
				readBytes += uint32(n)
			}
			body = &bytes.Buffer{}
			dataLen = binary.BigEndian.Uint32(lenBuf)
			for dataLen > 0 {
				if dataLen < 4096 {
					n, err = r.Read(dataBuf[:dataLen])
				} else {
					n, err = r.Read(dataBuf)
				}
				if err != nil {
					fmt.Printf("error: %v", err)
					break
					//c.cleanUp()
					// log if not eof client closes connection
				}
				body.Write(dataBuf[:n])
				dataLen -= uint32(n)
			}
		}
		err = h.Handle(opcode, data.Bytes(), body.Bytes())
		if err != nil {
			// do something?
			continue
		}
		if len(data.Bytes()) != 0 {
			data.Reset()
		}
		if len(body.Bytes()) != 0 {
			body.Reset()
		}
	}
}

// StartWriteLoop needs to be run before any Write calls are made.
func (c *Conn) StartWriteLoop(timeout time.Duration) {
	var p *Payload
	c.mu.Lock()
	if c.in == nil {
		c.in = make(chan *Payload, 100)
		c.queue = make(chan *Payload)
	}
	c.mu.Unlock()
	go c.proxyQueue()
	respLenBuf := make([]byte, 4)
	respDataBuf := new(bytes.Buffer)
	w := bufio.NewWriter(c.conn)
	encoder := NewEncoder(respDataBuf)
	var err error
	for p = range c.queue {
		if timeout != time.Duration(0) {
			c.conn.SetWriteDeadline(time.Now().Add(timeout))
		}
		err = w.WriteByte(uint8(p.Opcode))
		if err != nil {
			//log failed to write response + err
			//kill connection
			break
		}
		if p.Data != nil {
			err = encoder.Encode(p.Data)
			if err != nil {
				//log failed to encode response data + err
				break
			}
			binary.BigEndian.PutUint32(respLenBuf, uint32(respDataBuf.Len()))
			_, err = w.Write(respLenBuf)
			if err != nil {
				//log failed to write response + err
				//kill connection
				break
			}
			_, err = w.Write(respDataBuf.Bytes())
			if err != nil {
				//log failed to write response + err
				//kill connection
				break
			}
			respDataBuf.Reset()
		}
		if p.Body != nil {
			binary.BigEndian.PutUint32(respLenBuf, uint32(len(p.Body)))
			_, err = w.Write(respLenBuf)
			if err != nil {
				//log failed to write response + err
				//kill connection
				break
			}
			_, err = w.Write(p.Body)
			if err != nil {
				//log failed to write response + err
				//kill connection
				break
			}

		}
		err = w.Flush()
		if err != nil {
			break
			//log failed to write response + err
			//kill connection
		}
		// if err on write:
		//    c.Close()
		payloadPool.Put(p)
	}
}

func (c *Conn) proxyQueue() {
	var p *Payload
	in := c.in
	buf := []*Payload{}
	q := c.queue
	for {
		if len(buf) == 0 {
			p = <-in
			if p == nil {
				break
			}
			buf = append(buf, p)
		} else {
			select {
			case q <- buf[0]:
				buf = buf[1:]
			case p = <-in:
				if p == nil {
					break
				}
				buf = append(buf, p)
			}
		}
	}
}

// Handler cannot store 'data' it receives as it will change when new requests come in.
type Handler interface {
	Handle(opcode byte, data []byte, body []byte) error
}

func New(c net.Conn) *Conn {
	return &Conn{
		conn: c,
	}
}
