package GoCryptoTCP

import (
	"encoding/json"
)

var (
	ApplyCrypto  = 1
	AcceptCrypto = 2
	RejectCrypto = 3
	CryptoMsgRSA = 4
	CryptoMsgAES = 5
)

type Msg struct {
	MsgType   int    `json:"msgType"`
	From      int    `json:"from"`
	To        int    `json:"to"`
	Body      []byte `json:"body,omitempty"`
	RSAPubKey []byte `json:"RSAPubKey,omitempty"`
	AESKey    []byte `json:"AESKey,omitempty"`
	Sign      []byte `json:"sign,omitempty"`
}

func NewMsg(tp, fm, to int, body, RSAPubKey, AESKey, Sign []byte) *Msg {
	return &Msg{
		MsgType:   tp,
		From:      fm,
		To:        to,
		Body:      body,
		RSAPubKey: RSAPubKey,
		AESKey:    AESKey,
		Sign:      Sign,
	}
}

func NewUpgradeMsg(msgType, to int, body, RSAPubKey, AESKey []byte) *Msg {
	return &Msg{
		MsgType:   msgType,
		To:        to,
		Body:      body,
		RSAPubKey: RSAPubKey,
		AESKey:    AESKey,
	}
}

func NewCryptoMsg(msgType, from, to int, body []byte) *Msg {
	return &Msg{
		MsgType: msgType,
		From:    from,
		To:      to,
		Body:    body,
	}
}

func (m *Msg) Marshal() []byte {
	j, err := json.Marshal(m)
	CheckFatalErr(err)
	return j
}

func MsgUnmarshal(j []byte) (*Msg, error) {
	var m Msg
	err := json.Unmarshal(j, &m)
	return &m, err
}

func (m *Msg) String() string {
	return string(m.Marshal())
}

func (m *Msg) BodyString() string {
	return string(m.Body)
}
