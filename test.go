package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"time"
)

type TestConfig struct {
	RemoteIpAddr string `json:"remoteIpAddr"`
}

func getConfig() (*TestConfig, error) {
	return &TestConfig{
		RemoteIpAddr: "10.10.10.31",
	}, nil
}

var requestDefinitions = map[string][]byte{
	"get_printer_status": {
		0x54, 0x50, 0x52, 0x54, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x20,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	},
	// 00000000  74 70 72 74 00 00 00 00  00 00 00 01 00 00 00 20
	// 00000010  00 00 00 01 00 00 00 14  [IP       ] [?        ]
	// 00000020  14 00 00 XX 00 00 00 00  40 00 00 00 00 YY 00 00
	// 00000030  00 00 00 00
	// YY@+0x2d 00: printing, 01: done?

	"print_start": {
		0x54, 0x50, 0x52, 0x54, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x20,
		0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	},
	// 00000000  74 70 72 74 00 00 00 00  00 00 00 01 00 00 00 20  |tprt........... |
	// 00000010  00 00 00 02 00 00 00 03  00 00 00 00 00 00 00 00  |................|
	// 00000020  02 00 00

	"print_stop": {
		0x54, 0x50, 0x52, 0x54, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x20,
		0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	},
	// 00000000  74 70 72 74 00 00 00 00  00 00 00 01 00 00 00 20  |tprt........... |
	// 00000010  00 00 00 03 00 00 00 03  00 00 00 00 00 00 00 00  |................|
	// 00000020  03 00 00                                          |...|

	"request04": {
		0x54, 0x50, 0x52, 0x54, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x20,
		0x00, 0x00, 0x00, 0x04, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	},
	// 00000000  74 70 72 74 00 00 00 00  00 00 00 01 00 00 00 20  |tprt........... |
	// 00000010  00 00 00 04 00 00 00 03  00 00 00 00 00 00 00 00  |................|
	// 00000020  04 01 00                                          |...|

	"requestName": {
		0x54, 0x50, 0x52, 0x54, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x20,
		0x00, 0x00, 0x00, 0x05, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	},
	// 00000000  74 70 72 74 00 00 00 00  00 00 00 01 00 00 00 20  |tprt........... |
	// 00000010  00 00 00 05 00 00 00 40  00 00 00 00 00 00 00 00  |.......@........|
	// 00000020  54 45 50 52 41 20 50 52  4f 20 53 52 35 39 30 30  |TEPRA PRO SR5900|
	// 00000030  50 00 00 00 00 00 00 00  00 00 00 00 00 00 00 00  |P...............|
	// 00000040  53 52 35 39 30 30 50 41  32 38 41 37 36 00 00 00  |SR5900PA28A76...|
	// 00000050  00 00 00 00 00 00 00 00  00 00 00 00 00 00 00 00  |................|
}

func testUDPMessage(config *TestConfig, key string) []byte {
	conn, err := net.Dial("udp4", config.RemoteIpAddr+":9100")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	buffer := make([]byte, 1500)
	fmt.Printf("Sending %v\n", key)
	_, err = conn.Write(requestDefinitions[key])
	if err != nil {
		log.Fatal(err)
	}
	length, err := conn.Read(buffer)
	if err != nil {
		log.Fatal(err)
	}
	// fmt.Printf("%s", hex.Dump(buffer[:length]))
	return buffer[:length]
}

var header_common = []byte{
	0x1b, 0x7b, 0x07, 0x43, 0x02, 0x02, 0x01, 0x01, 0x49, 0x7d, // fixed
	0x1b, 0x7b, 0x03, 0x47, 0x47, 0x7d, // fixed
	0x1b, 0x7b, 0x04, 0x44, 0x05, 0x49, 0x7d, // fixed
	0x1b, 0x7b, 0x04, 0x73, 0x00, 0x73, 0x7d, // fixed
	0x1b, 0x7b, 0x05, 0x54, 0x40, 0x00, 0x94, 0x7d,
	0x1b, 0x7b, 0x05, 0x6c, 0x05, 0x05, 0x76, 0x7d, // fixed
	0x1b, 0x7b, 0x07, 0x7b, 0x00, 0x00, 0x53, 0x54, 0x22, 0x7d, // frame square? -> maybe not...
}
var header_print_size = []byte{
	0x1b, 0x7b, 0x07, 0x4c, 0xe2, 0x00, 0x00, 0x00, 0x2e, 0x7d, // 4c=data size, [4..8]: len?
}

// 360dpi =>
// px = mm/25.4*360
// mm = px/360*25.4

// 384dot =>
// 48*8 bits => 48 bytes per row is the max

var header_per_line = []byte{
	0x1b, 0x2e, 0x00, 0x0a, 0x0a, 0x01, 0x90, 0x00,
}
var termination = []byte{
	0x0c,
	0x1b, 0x7b, 0x03, 0x40, 0x40, 0x7d,
}

// 18mm : 1b2e 0000 0001 d700
// 12mm : 1b2e 000a 0a0a 9000

func testPrint(config *TestConfig) error {
	len_mm := 45.0
	len_px := int(len_mm * 360.0 / 25.4)
	log.Printf("len_px = 0x%08x", len_px)
	message_body := header_common
	binary.LittleEndian.PutUint32(header_print_size[4:], uint32(len_px))
	message_body = append(message_body, header_print_size...)
	fmt.Printf("%s", hex.Dump(message_body))
	for y := 0; y < len_px; y++ {
		content_line := make([]byte, 40)
		for i := 0; i < len(content_line); i++ {
			chunk := 0
			for k := 0; k < 8; k++ {
				x := i*8 + k
				t := x + y
				if t%32 == 0 {
					chunk = chunk | 15
				}
				chunk = chunk << 1
			}
			content_line[i] = byte(chunk)
		}
		line := append(header_per_line, content_line...)
		message_body = append(message_body, line...)
	}
	message_body = append(message_body, termination...)
	conn, err := net.Dial("tcp4", config.RemoteIpAddr+":9100")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	_, err = conn.Write(message_body)
	if err != nil {
		log.Fatal(err)
	}
	return nil
}

func do_print(config *TestConfig) {
	testUDPMessage(config, "print_start")
	testPrint(config)
	for {
		time.Sleep(500 * time.Millisecond)
		res := testUDPMessage(config, "get_printer_status")
		if res[0x2d] == 0 {
			log.Print("waiting...\n")
			continue
		}
		log.Printf("%02x\n", res[0x2d])
		break
	}
	testUDPMessage(config, "print_stop")
}

func get_tape_width(config *TestConfig) {
	res := testUDPMessage(config, "get_printer_status")
	expected := []byte{
		0x74, 0x70, 0x72, 0x74, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x20,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x14, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x14, 0x00, 0x06, 0x00, 0x00, 0x00, 0x00, 0x00, 0x40, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
	}
	fmt.Printf("%s", hex.Dump(res))
	for i := 0; i < len(expected); i++ {
		if i == 0x22 || i == 0x23 || i == 0x25 || i == 0x2d {
			continue
		}
		if res[i] != expected[i] {
			log.Fatalf("Unexpected byte at index 0x%x: expected %v but got 0x%02x", i, expected[i], res[i])
		}
	}

	index := 0x22
	if res[index] == 0x06 {
		log.Fatal("No tape!")
	}
	if res[index] == 0x21 {
		log.Fatal("Cover is open")
	}
	if res[index] != 0x00 {
		log.Fatalf("Unexpected byte at index 0x%02x: expected 0x00 or 0x06 but got 0x%02x", index, res[index])
	}

	index = 0x25
	if res[index] == 0x80 {
		log.Fatal("Cover is open!")
	}
	if res[index] != 0x00 {
		log.Fatalf("Unexpected byte at index 0x%02x: expected 0x00 or 0x06 but got 0x%02x", index, res[index])
	}

	index = 0x2d
	log.Printf("res[0x%02x] = 0x%02x", index, res[index])

	index = 0x23
	tape_index := res[index]
	if tape_index == 0x01 {
		fmt.Printf("Tape width = 6mm (tape_index = 0x%02x)\n", tape_index)
	} else if tape_index == 0x02 {
		fmt.Printf("Tape width = 9mm (tape_index = 0x%02x)\n", tape_index)
	} else if tape_index == 0x03 {
		fmt.Printf("Tape width = 12mm (tape_index = 0x%02x)\n", tape_index)
	} else if tape_index == 0x04 {
		fmt.Printf("Tape width = 18mm (tape_index = 0x%02x)\n", tape_index)
	} else if tape_index == 0x05 {
		fmt.Printf("Tape width = 24mm (tape_index = 0x%02x)\n", tape_index)
	} else if tape_index == 0x06 {
		fmt.Printf("Tape width = 36mm (tape_index = 0x%02x)\n", tape_index)
	} else {
		log.Fatalf("Unknown Tape width index: 0x%x\n", tape_index)
	}
}

func main() {
	config, err := getConfig()
	if err != nil {
		log.Fatal(err)
	}
	get_tape_width(config)
	do_print(config)
}
