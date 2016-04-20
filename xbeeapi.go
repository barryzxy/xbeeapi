package xbee

import (
	"github.com/coreyshuman/picontrol/serial"
	"time"
	"bufio"
	"bytes"
	"errors"
	"container/list"
)

// receive handler signature
type RxHandlerFunc func(string)

// rx handler struct
type RxHandler struct {
	id int
	name string
	RxHandlerFunc func
}

var rxHandlerList *list.List
quit := make(chan bool)

// https://golang.org/pkg/strings/
/*
var rxBuff string
var txBuff string
var rxReader *String.Reader
var txReader *String.Reader
*/
var rxBuffer bytes.Buffer
var txBuffer bytes.Buffer

////////////////////


func Init() {
	// rxReader = string.NewReader(rxBuff);
	// txReader = string.NewReader(txBuff);
}

func Begin() {
	go func() {
		for {
			select {
			case <- quit:
				return
			default:
				
			}
		}
		// if we get here, dispose and exit
	}
}

func End() {
	quit <- true
}

func processRxData()
{
	// use rxReader to parse out data
}

func processTxData()
{
	// send data out of serial (XBEE) port
}

func SendPacket(string address, string data) {
	
	// do a bunch of stuff
	
	txBuffer.WriteByte(byte('c'))
}


