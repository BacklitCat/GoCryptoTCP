package GoCryptoTCP

import (
	"bufio"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
)

type CryptoClient struct {
	rAddr *net.TCPAddr
	conn  *CryptoConn
}

func NewCryptoClient(severAddr string) *CryptoClient {
	rAddr, err := net.ResolveTCPAddr("tcp", severAddr)
	CheckFatalErr(err)
	return &CryptoClient{
		rAddr: rAddr,
	}
}

func (c *CryptoClient) Dial() *CryptoConn {
	conn, err := net.DialTCP("tcp", nil, c.rAddr)
	CheckFatalErr(err)
	c.conn = NewCryptoConn(conn)
	CheckFatalErr(c.conn.ApplyCrypto())
	return c.conn
}

func (c *CryptoClient) HandleConn() {
	for {
		m, err := c.conn.readMsg()
		if err != nil {
			if err == io.EOF {
				break
			}
			continue // discard
		}
		//log.Println("[readMsg]:", m.String())
		if m.MsgType >= CryptoMsgRSA {
			err = c.conn.DecryptMsg(m)
			if err != nil {
				continue // discard
			}
			//log.Println("[CryptoMsg]", m.From, "say:", m.BodyString())
			log.Println(m.From, "say:", m.BodyString())
		}

	}
	log.Println("stop handle connection")
}

func (c *CryptoClient) Scan() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		arr := strings.Fields(scanner.Text())
		if len(arr) == 0 {
			continue
		} else if arr[0] == "status" {
			log.Println(c.conn.String())
		} else if arr[0] == "close" {
			c.conn.Close()
			log.Println("[Conn Close]")
		} else if arr[0] == "to" {
			if len(arr) < 3 {
				log.Println("id or msg body is needed")
				continue
			}
			toId, err := strconv.Atoi(arr[1])
			if err != nil {
				log.Println(err)
				continue
			}
			CheckErr(c.conn.WriteCrypto(5, c.conn.CID, toId, strings.Join(arr[2:], " ")))
		}
	}
}
