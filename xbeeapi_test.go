package xbeeapi

import (
    "testing"
	"encoding/hex"
)

func TestSendPacket(t *testing.T) {
	d,n,e := SendPacket([]byte{0x00, 0x13, 0xa2, 0x00, 0x40, 0x0a, 0x01, 0x27}, nil, 0x00, []byte{0x54, 0x78, 0x44, 0x61, 0x74, 0x61, 0x30, 0x41})
	if e != nil {
		t.Error(e.Error())
	}
	t.Log(n)
	t.Log(hex.Dump(d))
}

func TestATCommand(t *testing.T) {
	d,n,e := SendATCommand([]byte{0x4e, 0x4a}, nil)
	if e != nil {
		t.Error(e.Error())
	}
	t.Log(n)
	t.Log(hex.Dump(d))
}

func TestParseATCommand(t *testing.T) {
	ft,d,e := ParseATCommandResponse([]byte{0x7e, 0x00, 0x05, 0x88, 0x01, 0x42, 0x44, 0x00, 0xf0})
	if e != nil {
		t.Error(e.Error())
	}
	t.Log(ft)
	t.Log(hex.Dump(d))
}

func TestParseATCommand2(t *testing.T) {
	ft,d,e := ParseATCommandResponse([]byte{0x7e, 0x00, 0x0a, 0x88, 0x01, 0x42, 0x44, 0x00, 0x68, 0x65, 0x6c, 0x6c, 0x6f, 0x4b})
	if e != nil {
		t.Error(e.Error())
	}
	t.Log(ft)
	t.Log(hex.Dump(d))
}