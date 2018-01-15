// Public Domain (-) 2018-present, The Elko Authors.
// See the Elko UNLICENSE file for details.

package op

// const (
// 	// Ops from service to elko
// 	Heartbeat = 1
// 	List      = 2 // list host:port for the specified service
// 	Next      = 3
// 	Hello     = 64
// 	Reloaded  = 65
// 	Completed = 66  // in response to shutdown
// 	Call      = 128 // service call to server
// 	Publish   = 129
// 	Response  = 130

// 	// Ops from elko to service
// 	Shutdown       = 1
// 	ListResponse   = 64
// 	PublishRequest = 65
// 	Request        = 66 // service called from server
// 	CallResponse   = 67
// 	Reload         = 68

// 	// Inter-node opcodes
// 	Hello           = 64
// 	Reserve         = 65
// 	ReserveResponse = 66
// 	Call            = 128
// 	Publish         = 129
// 	Response        = 130
// )

type Code uint8

const (
	// Client opcodes
	ClientHeartbeat = 1
	ClientHello     = 2
	ClientRequest   = 3
	ClientResponse  = 4
	ClientShutdown  = 5

	// Server opcodes
	ServerHello    = 64
	ServerRequest  = 65
	ServerShutdown = 66
)

var strings = map[Code]string{
	ClientHeartbeat: "ClientHeartbeat",
	ClientHello:     "ClientHello",
	ClientRequest:   "ClientRequest",
	ClientResponse:  "ClientResponse",
	ClientShutdown:  "ClientShutdown",
	ServerHello:     "ServerHello",
	ServerRequest:   "ServerRequest",
	ServerShutdown:  "ServerShutdown",
}

func (c Code) String() string {
	return strings[c]
}
