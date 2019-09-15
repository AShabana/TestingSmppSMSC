package main

// This app is concurrent for each session
/*
	This testing server will listen to multiple ports and  should gives you ctrl UI to :
		a. Disable/Enable enquire_link_resp to some sessions
		b. Disable/Enable submit_sm_resp to some sessions
		c. Reject/Accept binds with Perminant/Transient Rejection reason
			i.e. Auth Failed / Freeze the session
		XX . All above could be reduced to just implement [un]freeze this sessions and do [not] accept new binds
*/

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/fiorix/go-smpp/smpp"
	"github.com/fiorix/go-smpp/smpp/pdu"
	"github.com/google/uuid"
	//	"github.com/fiorix/go-smpp/smpp/smpptest"
	"github.com/fiorix/go-smpp/smpp/pdu/pdufield"
)

// type SmppHandler func(c smpp.Conn, m pdu.Body)
type SmppHandler func(c Conn, m pdu.Body)

type Server struct {
	SmscName            string
	Port                int
	freeze              bool
	acceptNewConnection bool
	Handler             SmppHandler
	Conns               []smpp.Conn
	mu                  sync.Mutex
	listener            net.Listener
}

func (srvr *Server) Start() {
	go srvr.Serve()
}

func (srvr *Server) Freeze() {
	log.Println("The value of isFreeze before exec is : ", srvr.freeze)
	log.Printf("Freezing (%s:%d)\n", srvr.SmscName, srvr.Port)
	srvr.freeze = true
}

func (srvr *Server) UnFreeze() {
	log.Println("The value of isFreeze before exec is : ", srvr.freeze)
	log.Printf("UnFreezing (%s:%d)\n", srvr.SmscName, srvr.Port)
	srvr.freeze = false
}

func (srvr *Server) UnBind() {
	var resp pdu.Body
	resp = pdu.NewUnbind()
	log.Printf("Unbinding all sessions for (%s:%d)\n", srvr.SmscName, srvr.Port)
	for idx, c := range srvr.Conns {
		c.Write(resp)
		log.Printf("\tUnBind (%s:%d) session/total (%d/%d) \n", srvr.SmscName, srvr.Port, idx, len(srvr.Conns))
		time.Sleep(10 * time.Second)
		c.Close()
	}
	srvr.Conns = srvr.Conns[:0]
	/* srvr.Conns = srvr.truncateDeletedSessions(deletedConn) */
}

/*
func (srvr *Server) truncateDeletedSessions(deletedConn []smpp.Conn) []smpp.Conn {
   deletedSet = mapset.NewSet()
   originalSet = mapset.NewSet()
   return originalSet.Difference(deletedSet)
}
*/

func (srvr *Server) Serve() {
	var c *conn
	for {
		log.Printf("Server Event loop ->  (%s:%d)\n", srvr.SmscName, srvr.Port)
		//###The code of we disable acccept new connection should be here
		clientConn, err := srvr.listener.Accept()
		if err != nil {
			log.Println("Could not accept new client connection !!!")
		}
		c = newConn(clientConn)
		log.Printf("newConn return----> c: %p\n", c)
		srvr.Conns = append(srvr.Conns, c)
		log.Println("----> srvr.Conns: ", srvr.Conns)
		log.Println("Spawning new sessios")
		log.Printf("----> c: %p\n", c)
		go srvr.smppSession(c) // original func name was handle(c)_
	}
}

// ###jThis one contain the events loop of this session so it must be connected to the http ctrl
func (srvr *Server) smppSession(c *conn) { //in original sample this was named as handle
	log.Println("Creating New Session Event loop")
	session_id := uuid.New()
	var _x bool
	_x = true
	done := make(chan bool)

	log.Println("smpp session id: ", session_id)
	log.Printf("----> c: %p\n", c)
	log.Println("----> srvr.Conns: ", srvr.Conns)
	log.Print("----> srvr.Conns: %p\n", srvr.Conns)
	ticker := time.NewTicker(10 * time.Second)
	defer c.Close()
	if err := srvr.auth(c); err != nil {
		log.Println("Auth fn error")
		log.Println(err)
		if err != io.EOF {
			log.Printf("server:[%d] auth failed \n", srvr.Port)
			log.Println(err)
		}
		return
	}
	go func() {
		for {
			select {
			case <-done:
				fmt.Println("Done ..", session_id)
                ticker.Stop()
				break
			case <-ticker.C:
				fmt.Println("Ticker check status", session_id)
				fmt.Println("var before c.Read = ", _x)
			}
		}
	}()
	for {
		log.Println("New request in the session ", session_id)
		log.Printf("----> c: %p\n", c)
		data, err := c.Read()
		_x = false
		if err != nil {
			log.Println("Error in read from conn")
			if err != io.EOF {
				log.Printf("server:[%d] Failed to read from the session", srvr.Port)
				log.Println(err)
			}
			if err == io.EOF {
				log.Println("Client Closed the session")
			}
			for i, v := range srvr.Conns {
				if v == c {
					srvr.Conns = append(srvr.Conns[:i], srvr.Conns[i+1:]...)
					var resp pdu.Body
					resp = pdu.NewUnbind()
					c.Write(resp)
					//c.Close() defer should handle this case
					done <- true
				}
			}
			break
		}
		log.Println("** Exec. srvr.Handler", srvr.Handler)
		srvr.Handler(c, data)
	}
    return
}

func (srvr *Server) auth(c *conn) error {
	log.Println("*** Inside aut fn ****")
	log.Printf("----> c: %p\n", c)
	packet, err := c.Read()
	if err != nil {
		return err
	}
	var resp pdu.Body
	switch packet.Header().ID {
	case pdu.BindTransmitterID:
		resp = pdu.NewBindTransmitterResp()
        resp.Header().Seq = packet.Header().Seq
	case pdu.BindReceiverID:
		resp = pdu.NewBindReceiverResp()
        resp.Header().Seq = packet.Header().Seq
	case pdu.BindTransceiverID:
		resp = pdu.NewBindTransceiverResp()
        resp.Header().Seq = packet.Header().Seq
	default:
		return errors.New("Rejected PDU as we expecting from new connection a Bind first")
	}
	fields := packet.Fields()
	user := fields[pdufield.SystemID]
	password := fields[pdufield.Password]
	if user == nil || password == nil {
		return errors.New("Empty or no systemId or password")
	}
	resp.Fields().Set(pdufield.SystemID, user)
	// Replace it with Switch case to allow the server to send diff. -ve Responses.
	if srvr.acceptNewConnection == true {
		log.Println("*** write auth response as acceptNewConnection is true")
		return c.Write(resp)
	}
	log.Println("*** write auth response as acceptNewConnection is False")
	log.Println("*** *****")
	return errors.New("Bind Disabled now")

}

// Should be replace with a func(port) return listener
func createListener(port int) net.Listener {
	log.Println("localhost:" + strconv.Itoa(port))
	listener, err := net.Listen("tcp", "localhost:"+strconv.Itoa(port))
	if err == nil {
		return listener
	} else {
		panic(fmt.Sprintf("smpptest: failed to listen on a port: %v", err))
	}
}

func NewDefaultServer(name string, port int) *Server {
	log.Println("creating listener for port " + strconv.Itoa(port))
	return &Server{
		SmscName:            name,
		Port:                port,
		freeze:              false,
		acceptNewConnection: true,
		Handler:             nil,
		Conns:               nil,
		listener:            createListener(port),
	}
}

func (srvr *Server) DefaultHandler(cli smpp.Conn, m pdu.Body) { //This is do nothing handler
	//func DefaultHandler(cli smpp.Conn, m pdu.Body) {
	// switch case for packet.Header.ID and handle submit_sm and enquire_link
	// unbind requensts (close(c))
	// if !srvr.freeze { response back } else do nothing
	log.Println("Inside the smpp handler fn DefaultHandler")
	if srvr.freeze {
		log.Println("Will not Respond back as this session freezed")
	} else {
		log.Printf("reading PDU : %#v", m)
		cli.Write(m)
	}
}

func (srvr *Server) SecondHandler(cli Conn, m pdu.Body) {
	//log.Println("Inside the smpp handler fn SecondHandler", m, cli)
	log.Println("Inside the smpp handler fn SecondHandler")
	//log.Printf("DEBUG: PDU from %s: %#v", cli.RemoteAddr(), m)
	//log.Println("END OF DEBUG")

	var response pdu.Body

	switch m.Header().ID {
	case pdu.SubmitSMID:
		log.Println("Receive new submit_sm from the client")
		msgID, _ := uuid.NewUUID()
		response = pdu.NewSubmitSMResp()
		//response.Header().Seq = m.Header().Seq++
		response.Header().Seq = m.Header().Seq
		response.Fields().Set(pdufield.MessageID, msgID.String())
		cli.Write(response)
	case pdu.EnquireLinkID:
		log.Println("Receive new enquire_link from the client")
		response = pdu.NewEnquireLinkResp()
		response.Header().Seq = m.Header().Seq
		cli.Write(response)
	case pdu.UnbindID:
		log.Println("Receive new unbind_req from the client")
		response = pdu.NewUnbindResp()
		log.Println("Sending Unbind")
		cli.Write(response)
		log.Println("Closing the session")
		cli.Close()
	default:
		log.Println("Un supported smpp command ", m.Header().ID)
		m.Header().Status = 0x00000003
		cli.Write(m)
	}
}
