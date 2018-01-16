// Public Domain (-) 2018-present, The Elko Authors.
// See the Elko UNLICENSE file for details.

package servicemanager

import (
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"

	"github.com/tav/elko/pkg/servicemanager/protocol"
	"github.com/tav/golly/log"
)

type service struct {
	sync.RWMutex
	closed   bool
	conn     net.Conn
	key      []byte
	outgoing [][]byte
	pending  chan []byte
	timeout  time.Duration
}

func (s *service) close() {
	s.Lock()
	s.conn.Close()
	s.closed = true
	s.Unlock()
}

func (s *service) opcodeError(opcode protocol.OP, err error) {
	log.Errorf("servicemanager: got error decoding %s: %s", opcode, err)
	s.close()
}

func (s *service) isClosed() bool {
	s.RLock()
	closed := s.closed
	s.RUnlock()
	return closed
}

func (s *service) read(buf []byte, size int) error {
	total := 0
	for {
		s.conn.SetReadDeadline(time.Now().Add(s.timeout))
		n, err := s.conn.Read(buf)
		if err != nil {
			if nerr, ok := err.(net.Error); ok {
				if nerr.Timeout() {
					continue
				}
			}
			return fmt.Errorf("servicemanager: got error when reading service connection: %s", err)
		}
		total += n
		if total == size {
			return nil
		}
	}
}

func (s *service) heartbeat() {
}

func (s *service) write(opcode protocol.OP, msg proto.Message) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		log.Errorf("servicemanager: got error encoding %s: %s", opcode, err)
		s.close()
		return err
	}
	dataLen := len(data)
	buf := make([]byte, 13+dataLen)
	buf[0] = byte(opcode)
	binary.BigEndian.PutUint32(buf[1:], uint32(dataLen))
	copy(buf[5:], data)
	s.pending <- buf
	return nil
}

func handleService(s *Server, conn net.Conn) {
	opcode := protocol.OP(0)
	dataBuf := make([]byte, 4096)
	dataLen := 0
	headerBuf := make([]byte, 5)
	hashBuf := make([]byte, 8)
	seen := false
	log.Info("Received client connection")
	svc := &service{
		conn:    conn,
		pending: make(chan []byte, 100),
		timeout: s.config.CallTimeout,
	}
	var err error
	for {
		err = svc.read(headerBuf, 5)
		if err != nil {
			log.Error(err)
			return
		}
		opcode = protocol.OP(headerBuf[0])
		if !seen {
			if opcode != protocol.OP_CLIENT_HELLO {
				log.Errorf("servicemanager: received %s when expecting CLIENT_HELLO as first message", opcode)
				svc.close()
				return
			}
		}
		dataLen = int(binary.BigEndian.Uint32(headerBuf[1:]))
		if dataLen > cap(dataBuf) {
			dataBuf = make([]byte, dataLen)
		}
		err = svc.read(dataBuf[:dataLen], dataLen)
		if err != nil {
			log.Error(err)
			return
		}
		err = svc.read(hashBuf, 8)
		if err != nil {
			log.Error(err)
			return
		}
		switch opcode {
		case protocol.OP_CLIENT_HEARTBEAT:
			svc.heartbeat()
		case protocol.OP_CLIENT_HELLO:
			msg := &protocol.ClientHello{}
			err := proto.Unmarshal(dataBuf[:dataLen], msg)
			if err != nil {
				svc.opcodeError(opcode, err)
				return
			}
			fmt.Println(msg)
		case protocol.OP_CLIENT_REQUEST:
			msg := &protocol.ClientRequest{}
			err := proto.Unmarshal(dataBuf[:dataLen], msg)
			if err != nil {
				svc.opcodeError(opcode, err)
				return
			}
		case protocol.OP_CLIENT_RESPONSE:
			msg := &protocol.ClientResponse{}
			err := proto.Unmarshal(dataBuf[:dataLen], msg)
			if err != nil {
				svc.opcodeError(opcode, err)
				return
			}
		case protocol.OP_CLIENT_SHUTDOWN:
			msg := &protocol.ClientShutdown{}
			err := proto.Unmarshal(dataBuf[:dataLen], msg)
			if err != nil {
				svc.opcodeError(opcode, err)
				return
			}
			svc.close()
			return
		default:
			log.Errorf("servicemanager: unknown opcode %d", opcode)
			svc.close()
			return
		}
	}
}

func isValidServiceID(id string) bool {
	if id == "" {
		return false
	}
	for _, char := range id {
		if (char >= 'a' && char <= 'z') || char == '.' || (char >= '0' && char <= '9') {
			continue
		}
		return false
	}
	return true
}
