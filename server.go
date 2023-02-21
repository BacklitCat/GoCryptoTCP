package GoCryptoTCP

import (
	"bufio"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"errors"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
)

var (
	MaxConnNum   = 1000
	StartConnID  = 1001
	MsgChanSize  = 64
	ConnExitSize = 8
)

var (
	CIDNotExist = errors.New("cid not exist")
)

type Server struct {
	lAddr          *net.TCPAddr
	listener       *net.TCPListener
	connMap        sync.Map
	msgChan        chan *Msg
	exitChan       chan *CryptoConn
	idPool         *IDPool
	LPriKey        *rsa.PrivateKey
	LPubKey        *rsa.PublicKey
	LPubKeyMarshal []byte
}

func NewServer(addr string) *Server {
	lAddr, err := net.ResolveTCPAddr("tcp", addr)
	CheckFatalErr(err)
	listener, err := net.ListenTCP("tcp", lAddr)
	CheckFatalErr(err)
	s := &Server{
		lAddr:    lAddr,
		listener: listener,
		exitChan: make(chan *CryptoConn, ConnExitSize),
		msgChan:  make(chan *Msg, MsgChanSize),
		idPool:   NewIDPool(MaxConnNum),
	}
	s.GenerateKey()
	return s
}

func (s *Server) GenerateKey() {
	priKey, err := rsa.GenerateKey(rand.Reader, RSABits)
	CheckFatalErr(err)
	s.LPriKey, s.LPubKey = priKey, &priKey.PublicKey
	s.LPubKeyMarshal = x509.MarshalPKCS1PublicKey(s.LPubKey)
}

func (s *Server) StartListen() {
	log.Println("[Start listening] Addr:", s.lAddr)
	go s.onClose()
	go s.onMsgMux()
	for {
		// Listen
		conn, err := s.listener.AcceptTCP()
		if err != nil {
			log.Println(err)
			continue
		}
		// handler
		go s.onAccept(conn)
	}
}

func (s *Server) onAssignID(conn *net.TCPConn) (*CryptoConn, error) {
	cid, err := s.idPool.Assign()
	if err != nil {
		return nil, err
	}
	cid += StartConnID
	cryptoConn := NewCryptoConn(conn)
	cryptoConn.CID = cid
	s.connMap.Store(cid, cryptoConn)
	return cryptoConn, err
}

func (s *Server) onAccept(conn *net.TCPConn) {
	cc, err := s.onAssignID(conn)
	if err != nil { // no available conn id
		cc.Close()
		return
	}
	log.Println("[New Conn] ID:", cc.CID, "Addr:", cc.RAddr())

	for {
		m, err := cc.readMsg()
		if err != nil {
			if err == io.EOF {
				break
			}
			continue // discard
		}
		m.From = cc.CID //prevent counterfeiting
		log.Println("[readMsg]:", m.String())
		s.msgChan <- m
	}
	s.exitChan <- cc
}

func (s *Server) onClose() {
	for cc := range s.exitChan {
		log.Println("[Conn Close] ID:", cc.CID, "Addr:", cc.RAddr())
		s.connMap.Delete(cc.CID)
	}
}

func (s *Server) GetConnNum() int {
	return s.idPool.using

}

func (s *Server) GetCCbyCID(cid int) (*CryptoConn, error) {
	ccIF, ok := s.connMap.Load(cid)
	if !ok {
		return nil, CIDNotExist
	}
	return ccIF.(*CryptoConn), nil
}

func (s *Server) GetCCbyCIDStr(cidStr string) (*CryptoConn, error) {
	cid, err := strconv.Atoi(cidStr)
	if err != nil {
		return nil, err
	}
	return s.GetCCbyCID(cid)
}

//func (s *Server) writeLine(cid int, data []byte) error {
//	cc, err := s.GetCCbyCID(cid)
//	if err != nil {
//		return err
//	}
//	return cc.writeLine(data)
//}

func (s *Server) onProcessCryptoApply(cc *CryptoConn, m *Msg) {
	rPubKey, err := x509.ParsePKCS1PublicKey(m.RSAPubKey)
	if err != nil {
		_ = cc.writeMsg(NewUpgradeMsg(RejectCrypto, cc.CID, []byte("wrong rsaPubKey"), nil, nil))
	}
	aesKey := GenAESKey()
	aesKey_, err := rsa.EncryptPKCS1v15(rand.Reader, rPubKey, aesKey)
	CheckFatalErr(err)
	err = cc.writeMsg(NewUpgradeMsg(AcceptCrypto, cc.CID, nil, s.LPubKeyMarshal, aesKey_))
	if err != nil { // write error
		return
	}
	cc.LPubKey, cc.RPubKey, cc.AESKey = s.LPubKey, rPubKey, aesKey
	cc.IsCrypto = true
}

func (s *Server) onToServerMsg(cc *CryptoConn, m *Msg) {
	err := cc.DecryptMsg(m)
	if err != nil {
		return // discard
	}
	log.Println("[CryptoMsg]", m.From, "say:", m.BodyString())
	m.From, m.To = ServerID, m.From // echo msg
	m.Body = append([]byte("the server receives your message: "), m.Body...)
	_ = cc.WriteCryptoMsg(m)
}

func (s *Server) onForwardMsg(cc *CryptoConn, m *Msg) error {
	ccToIF, ok := s.connMap.Load(m.To)
	if !ok {
		return CIDNotExist //discard
	}
	ccTo := ccToIF.(*CryptoConn)
	err := cc.DecryptMsg(m)
	if err != nil {
		return err // discard
	}
	err = ccTo.WriteCryptoMsg(m)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func (s *Server) onMsgMux() {
	for m := range s.msgChan {
		ccIF, ok := s.connMap.Load(m.From)
		if !ok {
			continue //discard
		}
		cc := ccIF.(*CryptoConn)
		if m.MsgType == ApplyCrypto {
			s.onProcessCryptoApply(cc, m)
		} else if m.MsgType >= CryptoMsgRSA {
			if m.To == ServerID {
				s.onToServerMsg(cc, m)
			} else {
				_ = s.onForwardMsg(cc, m) // discard wrong msg
			}
		}
	}
}

func (s *Server) Scan() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		arr := strings.Fields(scanner.Text())
		if len(arr) == 0 {
			continue
		} else if arr[0] == "status" {
			log.Println("Conn Num:", s.GetConnNum())
		} else if arr[0] == "show-all-conn" {
			s.connMap.Range(func(cid, cc any) bool {
				log.Println(cc.(*CryptoConn).String())
				return true
			})
		} else if arr[0] == "show-conn" {
			if len(arr) != 2 {
				log.Println("id is needed")
				continue
			}
			cc, err := s.GetCCbyCIDStr(arr[1])
			if err != nil {
				log.Println(err)
				continue
			}
			log.Println(cc.String())
		} else if arr[0] == "to" {
			if len(arr) < 3 {
				log.Println("id or msg body is needed")
				continue
			}
			cc, err := s.GetCCbyCIDStr(arr[1])
			if err != nil {
				log.Println(err)
				continue
			}
			err = cc.WriteCrypto(MSGCryptoType, ServerID, cc.CID, strings.Join(arr[2:], " "))
			if err != nil {
				log.Println(err)
				continue
			}
		}
	}
}
