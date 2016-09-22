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
	//"bufio"
	//"bytes"
	"errors"
	"encoding/hex"
)

const ATCOMMAND		= 0x08
const TRANSMIT		= 0x10
const ATRESPONSE	= 0x88
const MODEMSTATUS	= 0x8A
const TXSTATUS		= 0x8B
const RECEIVE		= 0x90
const EXPLICITRX	= 0x91

// extra debug output
var debug = true

// user defined callback functions declarations
type ATCommandCallbackFunc func(frameId byte, data []byte)
type ModemStatusCallbackFunc func(status byte)
type ReceivePacketCallbackFunc func(destinationAddress64 [8]byte, destinationAddress16 [2]byte, receiveOptions byte, receivedData []byte)


// user defined callback functions
var atCommandCallback ATCommandCallbackFunc = nil
var modemStatusCallback ModemStatusCallbackFunc = nil
var receivePacketCallback ReceivePacketCallbackFunc = nil


var quit chan bool 

var txBuf *srbuf.SimpleRingBuff
var rxBuf *srbuf.SimpleRingBuff

var errHandler func(error) = nil
var serialXBEE int = -1
var err error

var _frameId int = 1
var _running bool = false

////////////////////


func Init(dev string, baud int, timeout int) (int, error) {	
	txBuf = srbuf.Create(256)
	rxBuf = srbuf.Create(256)
	// initialize a serial interface to the xbee module
	serial.Init()
	serialXBEE, err = serial.Connect(dev, baud, timeout, true)
	
	quit = make(chan bool)

	return serialXBEE, err
}


func Begin() {
	if serialXBEE == -1 {
		return
	}
	
	_running = true
	go func() {
		for {
			select {
			case <- quit:
				break
			default:
				processRxData()
				processTxData()
				time.Sleep(time.Millisecond*15)
			}
		}
		// if we get here, dispose and exit
		serial.Disconnect(serialXBEE)
	}()
}

func Close() {
	if !_running && serialXBEE != -1 {
		serial.Disconnect(serialXBEE)
	}
	quit <- true
}

// Setup for user-defined callback functions

func SetupErrorHandler(f func(error)) {
	errHandler = f
}

func SetupATCommandCallback(f ATCommandCallbackFunc) {
	atCommandCallback = f
}

func SetupModemStatusCallback(f ModemStatusCallbackFunc) {
	modemStatusCallback = f
}

func SetupReceivePacketCallback(f ReceivePacketCallbackFunc) {
	receivePacketCallback = f
}

func processRxData() {
	var ret bool = false
	var frameType byte
	var frameId byte
	var status byte
	var destinationAddress64 [8]byte
	var destinationAddress16 [2]byte
	var receiveOptions byte
	var err error
	var d []byte
	var n int
	
	d = make([]byte, 256)
	n,err = serial.ReadBytes(serialXBEE, d)

	// cts todo - improve this
	if err == nil && n > 0 {
		
		for i:=0; i<n; i++ {
			rxBuf.PutByte(d[i])
			//fmt.Println(fmt.Sprintf("Read:[%02X]", d[i]))
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
		frameType = data[3]
		// parse api packet
		if debug {
			fmt.Println(fmt.Sprintf( "Rx frame=[%02X]", frameType))
		}
		switch frameType { // Frame Type
			case ATRESPONSE : 
				frameId, data, err = parseATCommandResponse(data)
			case MODEMSTATUS : 
				status, err = parseModemStatusResponse(data)
			case RECEIVE : 
				destinationAddress64, destinationAddress16, receiveOptions, data, err = parseReceivePacketResponse(data)
			default:
				err = errors.New("Frame Type not supported: [" + hex.Dump(data[3:4]) + "]")
		}
		// error handling
		if(err != nil) {
			if(errHandler != nil) {
				errHandler(err)
			}
			return
		}
		// send parsed data to user callbacks
		switch frameType { // Frame Type
			case ATRESPONSE : 
				if atCommandCallback != nil {
					atCommandCallback(frameId, data)
				}
			case MODEMSTATUS : 
				if modemStatusCallback != nil {
					modemStatusCallback(status)
				}
			case RECEIVE : 
				if receivePacketCallback != nil {
					receivePacketCallback(destinationAddress64, destinationAddress16, receiveOptions, data)
				}
		}
	}
}

func processTxData() {
	// send data out of serial (XBEE) port
	if txBuf.AvailByteCnt() > 0 {
		data := txBuf.GetBytes(0)
		serial.SendBytes(serialXBEE, data)
		if debug {
			fmt.Println(fmt.Sprintf( "Tx cnt=(%d)", len(data)))
		}
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
	d = []byte{0x7E, 0x00, 0x00, TRANSMIT, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
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
	d = []byte{0x7E, 0x00, 0x00, ATCOMMAND, 0x00, 0x00, 0x00}
	
	
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
func parseATCommandResponse(r []byte) (frameId byte, data []byte, err error) {
	err = nil
	if(r[3] != ATRESPONSE) {
		return 0, nil, errors.New("Invalid Frame Type") 
	}
	
	n := int(r[1])*256 + int(r[2])
	
	if(n != len(r) - 4) {
		return 0, nil, errors.New("Frame Length Error: " + fmt.Sprintf("%d, %d", n, len(r)-4)) 
	}
	
	check := CalcChecksum(r[3:n+3])
	if(check != r[n+3]) {
		return 0, nil, errors.New(fmt.Sprintf( "Checksum Error: calc=[%02X] read=[%02X]", check, r[n+3]))
	}
	
	// prepare return data
	frameId = r[4]

	if(n > 5) {
		data = r[8:3+n] // 8:8+n-5
	} else {
		data = nil
	}
	
	return
}

/* ***************************************************************
 * ParseModemStatusResponse
 * Parse a modem RF module status message
 *
 * 0		- Start Delimiter
 * 1-2		- Length
 * 3		- Frame Type (0x8A)
 * 4		- Status
 * 5		- checksum
 * ***************************************************************/
func parseModemStatusResponse(r []byte) (status byte, err error) {
	status = 0
	err = nil
	if(r[3] != MODEMSTATUS) {
		err = errors.New("Invalid Frame Type") 
		return
	}
	
	n := int(r[1])*256 + int(r[2])
	
	if(n != len(r) - 4) {
		err = errors.New("Frame Length Error: " + fmt.Sprintf("%d, %d", n, len(r)-4)) 
		return
	}
	
	check := CalcChecksum(r[3:n+3])
	if(check != r[n+3]) {
		err = errors.New(fmt.Sprintf( "Checksum Error: calc=[%02X] read=[%02X]", check, r[n+3]))
		return
	}
	
	// prepare return data
	status = r[4]
	return
}

/* ***************************************************************
 * GetModemStatusDescription
 * Convert modem status code to description string
 *
 * Input:
 *			status byte
 * Output:
 *			description string
 *
 * ***************************************************************/
func GetModemStatusDescription(status byte) (description string) {
	switch {
		case status == 0:
			return "Hardware reset"
		case status == 1:
			return "Watchdog timer reset"
		case status == 2:
			return "Joined network"
		case status == 3:
			return "Disassociated"
		case status == 6:
			return "Coordinator started"
		case status == 7:
			return "Network security key updated"
		case status == 0x0d:
			return "Voltage supply limit exceeded"
		case status == 0x11:
			return "Modem configuration changed while join in progress"
		case status >= 0x80:
			return "Stack error"
	}
	return "Unknown status"
}

/* ***************************************************************
 * ParseTransmitStatusResponse
 * Parse the TX request transmit status message.
 *
 * 0		- Start Delimiter
 * 1-2		- Length
 * 3		- Frame Type (0x8B)
 * 4		- Frame ID 
 * 5 - 6	- 16-bit address of destination
 * 7		- Transmit Retry Count
 * 8		- Delivery Status
 * 9		- Discovery Status
 * 10		- checksum
 * ***************************************************************/
func parseTransmitStatusResponse(r []byte) (frameId byte, destinationAddress [2]byte, retryCount byte, deliveryStatus byte, discoveryStatus byte, err error) {
	frameId = 0
	retryCount = 0
	deliveryStatus = 0
	discoveryStatus = 0
	err = nil
	
	if(r[3] != TXSTATUS) {
		err = errors.New("Invalid Frame Type") 
		return
	}
	
	n := int(r[1])*256 + int(r[2])
	
	if(n != len(r) - 4) {
		err = errors.New("Frame Length Error: " + fmt.Sprintf("%d, %d", n, len(r)-4)) 
		return
	}
	
	check := CalcChecksum(r[3:n+3])
	if(check != r[n+3]) {
		err = errors.New(fmt.Sprintf( "Checksum Error: calc=[%02X] read=[%02X]", check, r[n+3]))
		return
	}
	
	// prepare return data
	frameId = r[4]
	copy(destinationAddress[:], r[5:7])
	retryCount = r[7]
	deliveryStatus = r[8]
	discoveryStatus = r[9]
	
	return
}

/* ***************************************************************
 * GetDeliveryStatusDescription
 * Convert transmit delivery status code to description string
 *
 * Input:
 *			status byte
 * Output:
 *			description string
 *
 * ***************************************************************/
func GetDeliveryStatusDescription(status byte) (description string) {
	switch status {
		case 0x00:
			return "Success"
		case 0x01:
			return "MAC ACK failure"
		case 0x02:
			return "CCA failure"
		case 0x15:
			return "Invalid destination endpoint"
		case 0x21:
			return "Network ACK failure"
		case 0x22:
			return "Not joined to network"
		case 0x23:
			return "Self-addressed"
		case 0x24:
			return "Address not found"
		case 0x25:
			return "Route not found"
		case 0x26:
			return "Broadcast source failed to hear a neighbor relay the message"
		case 0x2b:
			return "Invalid binding table index"
		case 0x2c:
			return "Resource error (lack of free buffers, timers, etc)"
		case 0x2d:
			return "Attempted broadcast with APS transmission"
		case 0x2e:
			return "Attempted unicast with APS transmission, but EE=0"
		case 0x32:
			return "Resource error (lack of free buffers, timers, etc)"
		case 0x74:
			return "Data payload too large"
		case 0x75:
			return "Indirect message unrequested"
	}
	return "Unknown status"
}

/* ***************************************************************
 * GetDiscoveryStatusDescription
 * Convert transmit discovery status code to description string
 *
 * Input:
 *			status byte
 * Output:
 *			description string
 *
 * ***************************************************************/
func GetDiscoveryStatusDescription(status byte) (description string) {
	switch status {
		case 0x00:
			return "No discovery overhead"
		case 0x01:
			return "Address discovery"
		case 0x02:
			return "Route discovery"
		case 0x03:
			return "Address and route"
		case 0x40:
			return "Extended timeout discovery"	
	}
	return "Unknown status"
}

/* ***************************************************************
 * ParseReceivePacketResponse
 * Parse the RF data received packet
 *
 * 0		- Start Delimiter
 * 1-2		- Length
 * 3		- Frame Type (0x90)
 * 4 - 11   - 64-bit address of sender (MSB - LSB)
 * 12 - 13	- 16-bit address of sender
 * 14		- Receive options
 * 15 - n	- Received data
 * n+1		- checksum
 * ***************************************************************/
func parseReceivePacketResponse(r []byte) (destinationAddress64 [8]byte, destinationAddress16 [2]byte, receiveOptions byte, receivedData []byte, err error) {
	receiveOptions = 0
	err = nil
	
	if(r[3] != RECEIVE) {
		err = errors.New("Invalid Frame Type") 
		return
	}
	
	n := int(r[1])*256 + int(r[2])
	
	if(n != len(r) - 4) {
		err = errors.New("Frame Length Error: " + fmt.Sprintf("%d, %d", n, len(r)-4)) 
		return
	}
	
	check := CalcChecksum(r[3:n+3])
	if(check != r[n+3]) {
		err = errors.New(fmt.Sprintf( "Checksum Error: calc=[%02X] read=[%02X]", check, r[n+3]))
		return
	}
	
	// prepare return data
	copy(destinationAddress64[:], r[4:12])
	copy(destinationAddress16[:], r[12:14])
	receiveOptions = r[14]
	
	if(n > 12) {
		receivedData = r[15:3+n] // 15:15+n-12
	} 
	
	return
}

/* ***************************************************************
 * GetReceiveOptionDescription
 * Convert transmit discovery status code to description string
 *
 * Input:
 *			status byte
 * Output:
 *			description string
 *
 * ***************************************************************/
func GetReceiveOptionDescription(status byte) (description string) {
	var d [100]byte
	i := 0
	
	if status & 0x01 > 0 {
		i += copy(d[i:], []byte("Packet acknowledged."))
	}
	
	if status & 0x02 > 0 {
		i += copy(d[i:], []byte("Packet was a broadcast packet."))
	}
	
	if status & 0x20 > 0 {
		if i > 0 {
			d[i] = ' ';
			i++
		}
		i += copy(d[i:], []byte("Packet encrypted with APS encryption."))
	}
	
	if status & 0x40 > 0 {
		if i > 0 {
			d[i] = ' ';
			i++
		}
		i += copy(d[i:], []byte("Packet was sent from an end device."))
	}
	
	if i == 0 {
		return "Unkown receive options."
	} else {
		return string(d[0:i])
	}
}

/* ***************************************************************
 * ParseExplicitReceivePacketResponse
 * Parse the RF data received packet when explicit mode enabled
 *
 * 0		- Start Delimiter
 * 1-2		- Length
 * 3		- Frame Type (0x90)
 * 4 - 11   - 64-bit address of sender (MSB - LSB)
 * 12 - 13	- 16-bit address of sender
 * 14		- Source Endpoint
 * 15		- Destination Endpoint
 * 16 - 17  - Cluster ID the packet was addressed to
 * 18 - 19  - Profile ID the packet was addressed to
 * 20		- Receive options
 * 21 - n	- Received data
 * n+1		- checksum
 * ***************************************************************/
func parseExplicitReceivePacketResponse(r []byte) (destinationAddress64 [8]byte, destinationAddress16 [2]byte,
													sourceEndpoint byte, destinationEndpoint byte,
													clusterId [2]byte, profileId [2]byte,
													receiveOptions byte, receivedData []byte, err error) {
	sourceEndpoint = 0
	destinationEndpoint = 0
	receiveOptions = 0
	receivedData = nil
	err = nil
	
	if(r[3] != EXPLICITRX) {
		err = errors.New("Invalid Frame Type") 
		return
	}
	
	n := int(r[1])*256 + int(r[2])
	
	if(n != len(r) - 4) {
		err = errors.New("Frame Length Error: " + fmt.Sprintf("%d, %d", n, len(r)-4)) 
		return
	}
	
	check := CalcChecksum(r[3:n+3])
	if(check != r[n+3]) {
		err = errors.New(fmt.Sprintf( "Checksum Error: calc=[%02X] read=[%02X]", check, r[n+3]))
		return
	}
	
	// prepare return data
	copy(destinationAddress64[:], r[4:12])
	copy(destinationAddress16[:], r[12:14])
	sourceEndpoint = r[14]
	destinationEndpoint = r[15]
	copy(clusterId[:], r[16:18])
	copy(profileId[:], r[18:20])
	receiveOptions = r[20]
	
	if(n > 18) {
		receivedData = r[21:3+n] // 21:21+n-18
	} 
	
	return
}