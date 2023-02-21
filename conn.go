package GoCryptoTCP

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
)

var (
	//TCPReadBufSize      = 1024 // < 1460
	TCPReadBufSize      = 20 // < 1460
	RSABits             = 2048
	ServerID            = 1000
	transChanBuf        = 64
	LINEFEED       byte = 10
	MSGCryptoType       = CryptoMsgAES
)

type CryptoConn struct {
	c         *net.TCPConn
	CID       int
	IsLiving  bool
	IsCrypto  bool
	transChan chan []byte
	LPubKey   *rsa.PublicKey
	LPriKey   *rsa.PrivateKey
	RPubKey   *rsa.PublicKey
	AESKey    []byte
}

func NewCryptoConn(conn *net.TCPConn) *CryptoConn {
	cc := &CryptoConn{
		c:         conn,
		CID:       -1,
		IsLiving:  true,
		IsCrypto:  false,
		transChan: make(chan []byte, transChanBuf),
		LPubKey:   nil,
		LPriKey:   nil,
		RPubKey:   nil,
		AESKey:    nil,
	}
	go cc.readChan()
	return cc
}

func (cc *CryptoConn) String() string {
	return fmt.Sprintf("CID:%d, Living:%t, Crypto:%t, LAddr:%s, RAddr:%s",
		cc.CID, cc.IsLiving, cc.IsCrypto, cc.LAddr().String(), cc.RAddr().String())
}

func (cc *CryptoConn) LAddr() net.Addr {
	return cc.c.LocalAddr()
}

func (cc *CryptoConn) RAddr() net.Addr {
	return cc.c.RemoteAddr()
}

func (cc *CryptoConn) readChan() {
	var last []byte
	buf := make([]byte, TCPReadBufSize)
	var i, j int
	for cc.IsLiving {
		n, err := cc.c.Read(buf)
		if err != nil {
			cc.Close()
			return
		}
		for i, j = 0, 0; j < n; j++ {
			if buf[j] == LINEFEED {
				if last == nil {
					cc.transChan <- append([]byte{}, buf[i:j]...)
					i = j + 1
				} else {
					if j == 0 {
						cc.transChan <- append(last, buf[0:1]...)
						i, j = 1, 1
					} else {
						cc.transChan <- append(last, buf[i:j]...)
						i = j + 1
					}
					last = nil
				}
			}
		}
		if i != j {
			if last == nil {
				last = append([]byte{}, buf[i:j]...)
			} else {
				last = append(last, buf[i:j]...)
			}
		}
	}
}

func (cc *CryptoConn) readLine() ([]byte, error) {
	data, ok := <-cc.transChan
	if !ok {
		return nil, io.EOF
	}
	return data, nil
}

func (cc *CryptoConn) writeLine(data []byte) error {
	_, err := cc.c.Write(append(data, LINEFEED))
	return err
}

func (cc *CryptoConn) readMsg() (*Msg, error) {
	data, err := cc.readLine()
	if err != nil { // EOF
		return nil, err
	}
	m, err := MsgUnmarshal(data)
	if err != nil { // unmarshal error
		return nil, err
	}
	return m, nil
}

func (cc *CryptoConn) writeMsg(m *Msg) error {
	//log.Println("[WRITE MSG]", string(m.Marshal()))
	return cc.writeLine(m.Marshal())
}

func (cc *CryptoConn) DecryptMsg(m *Msg) error {
	var err error
	if m.MsgType == CryptoMsgRSA {
		m.Body, err = rsa.DecryptPKCS1v15(rand.Reader, cc.LPriKey, m.Body)
		if err != nil {
			return err
		}
	} else if m.MsgType == CryptoMsgAES {
		m.Body = AesDecryptCBC(cc.AESKey, m.Body)
	} else {
		return errors.New("wrong msg type to encrypt")
	}
	return err
}

func (cc *CryptoConn) WriteCryptoMsg(m *Msg) error {
	var err error
	if m.MsgType == CryptoMsgRSA {
		m.Body, err = rsa.EncryptPKCS1v15(rand.Reader, cc.RPubKey, m.Body)
		if err != nil {
			return err
		}
	} else if m.MsgType == CryptoMsgAES {
		m.Body = AesEncryptCBC(cc.AESKey, m.Body)
	} else {
		return errors.New("wrong msg type to encrypt")
	}
	return cc.writeMsg(m)
}

func (cc *CryptoConn) WriteCrypto(msgType, from, to int, body string) error {
	return cc.WriteCryptoMsg(NewCryptoMsg(msgType, from, to, []byte(body)))
}

func (cc *CryptoConn) Close() {
	err := cc.c.Close()
	CheckFatalErr(err)
	close(cc.transChan)
	cc.IsLiving = false
}

func (cc *CryptoConn) GenRsaKey() {
	priKey, err := rsa.GenerateKey(rand.Reader, RSABits)
	CheckFatalErr(err)
	cc.LPriKey, cc.LPubKey = priKey, &priKey.PublicKey
}

func (cc *CryptoConn) LPubKeyMarshal() []byte {
	return x509.MarshalPKCS1PublicKey(cc.LPubKey)
}

func (cc *CryptoConn) ApplyCrypto() error {
	if cc.IsCrypto == true {
		return errors.New("already crypto")
	}

	applyRes := make(chan bool, 1)
	responseMsg := make(chan *Msg, 2)
	cc.GenRsaKey()
	var err error
	err = cc.writeMsg(NewUpgradeMsg(ApplyCrypto, ServerID, nil, cc.LPubKeyMarshal(), nil))
	if err != nil {
		return err
	}

	go func() {
		for {
			m, err := cc.readMsg()
			if err != nil {
				if err == io.EOF {
					applyRes <- false
					break
				}
				continue // wrong unmarshal, discard
			}
			responseMsg <- m
			//log.Println(m.String())
			break
		}
	}()

	for {
		select {
		case b := <-applyRes:
			if b {
				log.Println("[Encrypt Success] your connection is encrypted now")
				log.Println("[ID] your ID is:", cc.CID)
				return nil
			}
			return errors.New("[Encrypt Failed] connection closed")
		case m := <-responseMsg:
			if m.MsgType == AcceptCrypto {
				cc.CID = m.To
				cc.RPubKey, err = x509.ParsePKCS1PublicKey(m.RSAPubKey)
				CheckFatalErr(err)
				cc.AESKey, err = cc.LPriKey.Decrypt(rand.Reader, m.AESKey, nil)
				CheckFatalErr(err)
				cc.IsCrypto = true
				applyRes <- true
			} else if m.MsgType == RejectCrypto {
				return errors.New("[Encrypt Failed] server rejects application")
			} else {
				return errors.New(fmt.Sprintf("[Encrypt Failed] unhandled msg type: %d", m.MsgType))
			}
		}
	}
}
