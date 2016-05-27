package xbeeapi
/* XBEE API Mode Library
 * http://github.com/coreyshuman/xbeeapi
 * (C) 2016 Corey Shuman
 * 5/26/16
 *
 * License: MIT
 */

import (
	"github.com/coreyshuman/serial"
	"github.com/coreyshuman/srbuf"
	"time"
	"fmt"
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

var txBuf srbuf.SimpleRingBuff
var rxBuf srbuf.SimpleRingBuff

var serialXBEE int = -1
var err error

////////////////////


func Init(dev string, baud int, timeout int) {	
	txBuf = srbuf.Create(256)
	rxBuf = srbuf.Create(256)
	// initialize a serial interface to the xbee module
	serialXBEE, err = serial.Connect(dev, baud, timeout)
}

func Begin() {
	if serialXBEE == -1 {
		return
	}
	
	go func() {
		for {
			select {
			case <- quit:
				return
			default:
				
			}
		}
		// if we get here, dispose and exit
		serial.Disconnect(serialXBEE)
	}
}

func End() {
	quit <- true
}

func processRxData()
{
	// use rxBuf to parse out data
}

func processTxData()
{
	// send data out of serial (XBEE) port
	if txBuf.AvailableByteCount() > 0 {
		data := txBuf.GetBytes()
		serial.SendBytes(serialXBEE, data)
	}
}

func SendPacket(address byte[], data byte[], length int) {
	
	// do a bunch of stuff
	
	err := txBuffer.WriteByte(byte('c'))
	if err != nil {
		fmt.Print(err)
	}
}

func CalcChecksum()
{

}




