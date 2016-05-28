package xbeeapi

import (
    "testing"
	"encoding/hex"
)

func TestSendPacket(t *testing.T) {
	d,n,e := SendPacket([]byte{0x00, 0x13, 0xa2, 0x00, 0x40, 0x0a, 0x01, 0x27}, nil, 0x00, []byte{0x54, 0x78, 0x44, 0x61, 0x74, 0x61, 0x30, 0x41})
	if e != nil {
		t.Log(e.Error())
	}
	t.Log(n)
	t.Log(hex.Dump(d))
}

func TestATCommand(t *testing.T) {
	d,n,e := SendATCommand([]byte{0x4e, 0x4a}, nil)
	if e != nil {
		t.Log(e.Error())
	}
	t.Log(n)
	t.Log(hex.Dump(d))
}