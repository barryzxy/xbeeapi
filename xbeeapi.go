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
	//"time"
	"fmt"
	//"bufio"
	//"bytes"
	"errors"
	"container/list"
)

// receive handler signature
type RxHandlerFunc func(string)

// rx handler struct
type RxHandler struct {
	id int
	name string
	// RxHandlerFunc func
}

var rxHandlerList *list.List
var quit chan bool 

var txBuf *srbuf.SimpleRingBuff
var rxBuf *srbuf.SimpleRingBuff

var serialXBEE int = -1
var err error

////////////////////


func Init(dev string, baud int, timeout int) {	
	txBuf = srbuf.Create(256)
	rxBuf = srbuf.Create(256)
	// initialize a serial interface to the xbee module
	serialXBEE, err = serial.Connect(dev, baud, timeout)
	quit = make(chan bool)
}

/*
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
				return // cts debug
			}
		}
		// if we get here, dispose and exit
		serial.Disconnect(serialXBEE)
	}()
}
*/
/*
func End() {
	quit <- true
}

func processRxData()
{
	if true {
	
	}
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

/* ***************************************************************
 * Send data packet as an RF packet to the specified destination
 *
 * 0		- Start Delimiter
 * 1-2		- Length
 * 3		- Frame Type (0x10)
 * 4		- Frame ID 
 * 5 - 12	- 64-bit address MSB-LSB
 * 13 - 14	- 16-bit address MSB-LSB
 * 15		- broadcast radius
 * 16		- options
 * 17 - n	- RF data payload
 * n+1		- checksum
 * ***************************************************************/
func SendPacket(address64 []byte, address16 []byte, option byte, data []byte) (d []byte, n int, err error) {
	d = []byte{0x7E, 0x00, 0x00, 0x10, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0xFF, 0xFE, 0x00, 0x00}
	
	
	if len(address64) != 8 {
		return d, 0, errors.New("Incorrect Address Length")
	}
	
	// 64-bit address
	copy(d[5:13], address64)
	
	if address16 != nil && len(address16) == 2 {
		copy(d[13:15], address16)
	}
	
	d[16] = option
	
	d = append(d[:], data[:]...)
	d = append(d[:], 0x00)
	
	n = len(d)
	d[1] = byte((n-4) / 0x100)
	d[2] = byte((n-4) % 0x100)
	
	d[n-1] = CalcChecksum(d[3:])
	fmt.Println(string(d))
	return
}

func CalcChecksum(data []byte)(byte) {
	n := len(data)
	var cs byte = 0
	
	for i := 0; i < n-1; i++ {
		cs += data[i]
	}
	return 0xFF - cs
}




