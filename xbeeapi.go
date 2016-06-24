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
	"encoding/hex"
)

// receive handler signature
type RxHandlerFunc func([]byte)

// rx handler struct
type RxHandler struct {
	name string
	frameType byte
	handlerFunc func([]byte)
}

var rxHandlerList *list.List
var quit chan bool 

var txBuf *srbuf.SimpleRingBuff
var rxBuf *srbuf.SimpleRingBuff

var errHandler func(error) = nil
var serialXBEE int = -1
var err error

var _frameId int = 1

////////////////////


func Init(dev string, baud int, timeout int) (int, error) {	
	txBuf = srbuf.Create(256)
	rxBuf = srbuf.Create(256)
	// initialize a serial interface to the xbee module
	serial.Init()
	serialXBEE, err = serial.Connect(dev, baud, timeout)
	quit = make(chan bool)
	rxHandlerList = list.New()
	return serialXBEE, err
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
				processRxData()
				processTxData()
			}
		}
		// if we get here, dispose and exit
		serial.Disconnect(serialXBEE)
	}()
}


func End() {
	quit <- true
}

// cts todo - avoid repeat for same framdId
func AddHandler(frameType byte, f func([]byte)) {
	var handler RxHandler
	handler.name = "test"
	handler.frameType = frameType
	handler.handlerFunc = f
	rxHandlerList.PushBack(handler)
}

func findHandler(frameType byte) RxHandlerFunc {
	for e := rxHandlerList.Front(); e != nil; e = e.Next() {
		if e.Value.(RxHandler).frameType == frameType {
			return e.Value.(RxHandler).handlerFunc
		}
	}
	return nil
}

func SetupErrorHandler(f func(error)) {
	errHandler = f
}

func processRxData() {
	var ret bool = false
	var frameId byte
	var err error
	var d []byte
	var n int
	
	d = make([]byte, 256)
	n,err = serial.ReadBytes(serialXBEE, d)
	// cts todo - improve this
	if err == nil && n > 0 {
		
		for i:=0; i<n; i++ {
			rxBuf.PutByte(d[i])
			fmt.Println(fmt.Sprintf("Read:[%02X]", d[i]))
		}
	}
	
	for !ret {
		avail := rxBuf.AvailByteCnt()
		if(avail < 8) { // 8 bytes is minimum for complete packet
			break
		}
		p := rxBuf.PeekBytes(3)
		if(p[0] != 0x7E) {
			rxBuf.GetByte() // skip byte, increment buffer
			continue
		}
		n := int(p[1])*256 + int(p[2])
		if(avail < n+4) { // not all data received yet, break for now
			break
		}
		ret = true
		// if we get here, packet is ready to parse
		data := rxBuf.GetBytes(n+4)
		switch(data[3]) { // Frame Type
			case 0x88 : // at command response
				frameId, data, err = ParseATCommandResponse(data)
				break
				
			default:
				err = errors.New("Frame Type not supported: [" + hex.Dump(data[3:4]) + "]")
				break
		}
		if(err != nil) {
			if(errHandler != nil) {
				errHandler(err)
			}
			return
		}
		// fire callback
		handler := findHandler(frameId) 
		if(handler != nil) {
			handler(data)
		} else {
			fmt.Println("No Handler")
		}
	}
}

func processTxData() {
	// send data out of serial (XBEE) port
	if txBuf.AvailByteCnt() > 0 {
		data := txBuf.GetBytes(0)
		serial.SendBytes(serialXBEE, data)
	}
}

/* ***************************************************************
 * SendPacket
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
	d = []byte{0x7E, 0x00, 0x00, 0x10, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
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
	
	// cts todo - improve this
	for i := 0; i<len(d); i++ {
		txBuf.PutByte(d[i])
	}

	return
}

/* ***************************************************************
 * SendATCommand
 * Send AT command to the local device and apply changes immediately
 *
 * 0		- Start Delimiter
 * 1-2		- Length
 * 3		- Frame Type (0x08)
 * 4		- Frame ID 
 * 5 - 6	- AT command
 * 			- optional parameter
 * 7		- checksum
 * ***************************************************************/
func SendATCommand(command []byte, param []byte) (d []byte, n int, err error) {
	d = []byte{0x7E, 0x00, 0x00, 0x08, 0x00, 0x00, 0x00}
	
	
	if len(command) != 2 {
		return d, 0, errors.New("Incorrect AT Command Length")
	}
	
	d[4] = byte(_frameId)
	_frameId ++ // cts todo - make this better
	d[5] = command[0]
	d[6] = command[1]
	
	// copy param if exists
	d = append(d[:], param...)
	d = append(d[:], 0x00)
	
	n = len(d)
	d[1] = byte((n-4) / 0x100)
	d[2] = byte((n-4) % 0x100)
	
	d[n-1] = CalcChecksum(d[3:])
	
	// cts todo - improve this
	for i := 0; i<len(d); i++ {
		txBuf.PutByte(d[i])
	}

	return
}

func CalcChecksum(data []byte)(byte) {
	n := len(data)
	var cs byte = 0

	for i := 0; i < n; i++ {
		cs += data[i]
	}
	return 0xFF - cs
}


/* ***************************************************************
 * ParseATCommandResponse
 * Parse an AT Command response from XBEE
 *
 * 0		- Start Delimiter
 * 1-2		- Length
 * 3		- Frame Type (0x88)
 * 4		- Frame ID 
 * 5 - 6	- AT command
 * 			- optional command data
 * 7		- checksum
 * ***************************************************************/
func ParseATCommandResponse(r []byte) (frameId byte, data []byte, err error) {
	err = nil
	if(r[3] != 0x88) {
		return 0, nil, errors.New("Invalid Frame Type") 
	}
	
	n := int(r[1])*256 + int(r[2])
	
	if(n != len(r) - 4) {
		return 0, nil, errors.New("Frame Length Error: " + fmt.Sprintf("%d, %d", n, len(r)-4)) 
	}
	
	check := CalcChecksum(r[3:n+3])
	if(check != r[n+3]) {
		return 0, nil, errors.New(fmt.Sprintf( "Checksum Error: calc=[%02X] read=[%02X]", check, r[n+3] ) )
	}
	
	// prepare return data
	frameId = r[3]

	if(n > 5) {
		data = r[8:3+n] // 8:8+n-5
	} else {
		data = nil
	}
	
	return
}




