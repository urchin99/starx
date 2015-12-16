package starx

import (
	"encoding/json"
	"errors"
	"fmt"
	"starx/rpc"
	"time"
)

type SessionStatus byte

const (
	_ SessionStatus = iota
	SS_START
	SS_HANDSHAKING
	SS_WORKING
	SS_CLOSED
)

var (
	ErrRPCLocal = errors.New("RPC object must location in different server type")
)

// This session type as argument pass to Handler method, is a proxy session
// for frontend session in frontend server or backend session in backend
// server, correspond frontend session or backend session id as a field
// will be store in type instance
//
// This is user sessions, not contain raw sockets information
type Session struct {
	Id           uint64        // session global uniqe id
	Uid          int           // binding user id
	reqId        uint          // last request id
	status       SessionStatus // session current time
	lastTime     int64         // last heartbeat time
	rawSessionId uint64        // raw session id, frontendSession in frontend server, or backendSession in backend server
}

// Create new session instance
func newSession() *Session {
	return &Session{
		Id:       connectionService.getNewSessionUUID(),
		status:   SS_START,
		lastTime: time.Now().Unix()}
}

// Session send packet data
func (session *Session) Send(data []byte) {
	Net.send(session, data)
}

// Push message to session
func (session *Session) Push(route string, data []byte) {
	if App.Config.IsFrontend {
		Net.Push(session, route, data)
	} else {
		rs, err := Net.getRemoteSessionBySid(session.rawSessionId)
		if err != nil {
			Error(err.Error())
		} else {
			sid, ok := rs.bsessionIdMap[session.Id]
			if !ok {
				Error("sid not exists")
				return
			}
			resp := rpc.Response{}
			resp.Route = route
			resp.ResponseType = rpc.RPC_HANDLER_PUSH
			resp.Reply = data
			resp.Sid = sid
			writeResponse(rs, &resp)
		}
	}
}

// Response message to session
func (session *Session) Response(data []byte) {
	if App.Config.IsFrontend {
		Net.Response(session, data)
	} else {
		rs, err := Net.getRemoteSessionBySid(session.rawSessionId)
		if err != nil {
			Error(err.Error())
		} else {
			sid, ok := rs.bsessionIdMap[session.Id]
			if !ok {
				Error("sid not exists")
				return
			}
			resp := rpc.Response{}
			resp.ResponseType = rpc.RPC_HANDLER_RESPONSE
			resp.Reply = data
			resp.Sid = sid
			writeResponse(rs, &resp)
		}
	}
}

func (session *Session) Bind(uid int) {
	if session.Uid > 0 {
		session.Uid = uid
	} else {
		Error("uid invalid")
	}
}

func (session *Session) String() string {
	return fmt.Sprintf("Id: %d, Uid: %d, RemoteAddr: %s",
		session.Id,
		session.Uid)
}

func (session *Session) heartbeat() {
	session.lastTime = time.Now().Unix()
}

func (session *Session) AsyncRPC(route string, args ...interface{}) error {
	ri, err := decodeRouteInfo(route)
	if err != nil {
		return err
	}
	encodeArgs, err := json.Marshal(args)
	if err != nil {
		return err
	}
	if App.Config.Type == ri.server {
		return ErrRPCLocal
	} else {
		remote.request("user", ri, session, encodeArgs)
		return nil
	}
}

func (session *Session) RPC(route string, args ...interface{}) error {
	return session.AsyncRPC(route, args)
}

// Sync session setting to frontend server
func (session *Session) Sync(string) {
	//TODO
	//synchronize session setting field to frontend server
}

// Sync all settings to frontend server
func (session *Session) SyncAll() {
}
