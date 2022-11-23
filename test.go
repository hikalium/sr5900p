package main

import (
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

/*
$ tshark -Y 'ip.addr == 10.10.10.31 && udp' -r dump_W18mm_L30mm_2.bin -T fields -e data | grep -E '(74707274|54505254)(00000000)(00000001)(00000020)' | sed -E 's/^.{36}(.{4})(.{8})(.{8})(.{4})(.{4})(.*$)/\1 \2 \3 \4 \5 \6/'| sort | uniq

cmd  datalen  ipv4addr job  ??
0001 00000000 0a0a0a5a a397 cd03
0001 00000000 0a0a0a5a a397 d02c
0001 00000000 0a0a0a5a a397 d41d
0001 00000000 0a0a0a5a a397 d81d
0001 00000000 0a0a0a5a a397 d86a
0001 00000014 0a0a0a5a a397 cd03 1400000400000000400000000001000000000000
0001 00000014 0a0a0a5a a397 d02c 1400000400000000400000000001000000000000
0001 00000014 0a0a0a5a a397 d41d 1400000400000000400000000001000000000000
0001 00000014 0a0a0a5a a397 d81d 1400000400000000400000000001000000000000
0001 00000014 0a0a0a5a a397 d86a 1400000400000000400000000001000000000000
0001 00000014 0a0a0a5a a397 d86a 1402000400000000400000000000000000000000
0002 00000000 0a0a0a5a a397 d86a
0002 00000003 0a0a0a5a a397 d86a 020000
0003 00000000 0a0a0a5a a397 d86a
0003 00000003 0a0a0a5a a397 d86a 030000
0004 00000000 0a0a0a5a a397 d845
0004 00000003 0a0a0a5a a397 d845 040100
0100 00000000 0a0a0a5a a397 d86a
0100 00000001 0a0a0a5a a397 d86a 10
0101 00000000 0a0a0a5a a397 d86a
*/

var requestDefinitions = map[string][]byte{
	"get_printer_status": {
		// cmd == 0x01
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
		// cmd == 0x02
		0x54, 0x50, 0x52, 0x54, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x20,
		0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	},
	// 00000000  74 70 72 74 00 00 00 00  00 00 00 01 00 00 00 20  |tprt........... |
	// 00000010  00 00 00 02 00 00 00 03  00 00 00 00 00 00 00 00  |................|
	// 00000020  02 00 00

	"print_stop": {
		// cmd == 0x03
		0x54, 0x50, 0x52, 0x54, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x20,
		0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	},
	// 00000000  74 70 72 74 00 00 00 00  00 00 00 01 00 00 00 20  |tprt........... |
	// 00000010  00 00 00 03 00 00 00 03  00 00 00 00 00 00 00 00  |................|
	// 00000020  03 00 00                                          |...|

	"request04": {
		// cmd == 0x04
		0x54, 0x50, 0x52, 0x54, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x20,
		0x00, 0x00, 0x00, 0x04, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	},
	// 00000000  74 70 72 74 00 00 00 00  00 00 00 01 00 00 00 20  |tprt........... |
	// 00000010  00 00 00 04 00 00 00 03  00 00 00 00 00 00 00 00  |................|
	// 00000020  04 01 00                                          |...|

	"requestName": {
		// cmd == 0x05
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
	"request0100": {
		// cmd == 0x0100
		0x54, 0x50, 0x52, 0x54, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x20,
		0x00, 0x00, 0x00, 0x05, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	},
	// ================================================ [IP    ] == [job?]
	// 545052540000000000000001000000200000010000000000 0a0a0a5a a3 6bc80f
	// 747072740000000000000001000000200000010000000001 0a0a0a5a a3 6bc80f 10
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
	return buffer[:length]
}

var header_common = []byte{
	//
	0x1b, 0x7b, 0x07, 0x7b, 0x00, 0x00, 0x53, 0x54, 0x22, 0x7d,
	0x1b, 0x7b, 0x07, 0x43, 0x02, 0x02, 0x01, 0x01, 0x49, 0x7d,
	0x1b, 0x7b, 0x04, 0x44, 0x05, 0x49, 0x7d,
	0x1b, 0x7b, 0x03, 0x47, 0x47, 0x7d,
	0x1b, 0x7b, 0x04, 0x73, 0x00, 0x73, 0x7d,
	0x1b, 0x7b, 0x05, 0x6c, 0x05, 0x05, 0x76, 0x7d,
}

/*
1b 7b 07 4c 4c __ __ 00 00 __ 7d 1b 7b 05 54 __ 00 __ 7d common
           L = a9 01       f6 = 30mm
           L = 35 02       83 = 40mm
           L = c1 02       0f = 50mm
           L = 87 05       d8 = 100mm
           L = 13 06       65 = 110mm
           L = a4 06       f6 = 120mm
                                             91    e5    ./dump_W12mm_L110mm.bin
                                             91    e5    ./dump_W12mm_L50mm.bin
                                             92    e6    ./dump_W12mm_L100mm.bin
                                             92    e6    ./dump_W12mm_L40mm.bin
                                             93    e7    ./dump_W12mm_L120mm.bin
                                             93    e7    ./dump_W12mm_L30mm.bin
                                             94    e8    ./dump_W18mm_L50mm.bin
                                             95    e9    ./dump_W18mm_L40mm.bin
                                             96    ea    ./dump_W18mm_L30mm.bin
                                             96    ea    ./dump_W18mm_L30mm_2.bin
                                             96    ea    ./dump_W24mm_L110mm.bin
                                             96    ea    ./dump_W24mm_L110mm_2.bin
*/
var header_print_size = []byte{
	// Tape Length
	//0x1b, 0x7b, 0x07, // common
	//
	//0x4c, 0xa9, 0x01, 0x00, 0x00, 0x00, 0x7d, // L=30mm
	//0x4c, 0x35, 0x02, 0x00, 0x00, 0x00, 0x7d, // L=40mm
	//0x4c, 0x35, 0x00, 0x00, 0x00, 0x00, 0x7d,
	// 4c AA AA 00 00 00 7d
	//    AA AA = L [mm] * 25.4 / 360 = length in px
	//               (BB, L) = (216, 100), (15,50), (131, 40)
	// 0x4c, 0x35, 0x02, 0x00, 0x00, 0x83, 0x7d, // L=40mm
	// 4c c10200000f 7d for L=50mm

	// ????
	0x1b, 0x7b, 0x05,
	//0x54, 0x96, 0x00, 0xea, 0x7d, // W=18mm, L=30mm
	// 0x54, 0x95, 0x00, 0xe9, 0x7d, // W=18mm, L=40mm
	// 0x54, 0x94, 0x00, 0xe8, 0x7d, // W=18mm, L=50mm
	//0x54, 0x93, 0x00, 0xe7, 0x7d, // W=12mm, L=30mm
	// 0x54, 0x92, 0x00, 0xe6, 0x7d, // W=12mm, L=40mm
	0x54, 0x91, 0x00, 0xe5, 0x7d, // W=12mm, L=50mm
}

// 360dpi =>
// px = mm/25.4*360
// mm = px/360*25.4

// 384dot =>
// 48*8 bits => 48 bytes per row is the max

var header_per_line = []byte{
	// 1b2e00000001 [width_in_px: u16]
	0x1b, 0x2e, 0x00, 0x00, 0x00, 0x01, 0x1d, 0x01,
}
var termination = []byte{
	// constant
	0x0c,
	0x1b, 0x7b, 0x03, 0x40, 0x40, 0x7d,
}

// 18mm : 1b2e 0000 0001 d700
// 12mm : 1b2e 000a 0a0a 9000

func testPrint(config *TestConfig) error {
	w_px := 384
	w_bytes := (w_px + 7) / 8
	log.Printf("w_bytes = %v", w_bytes)
	len_mm := 40.0
	len_px := int(len_mm * 360.0 / 25.4)
	log.Printf("len_px = 0x%08x", len_px)
	message_body := header_common
	//binary.LittleEndian.PutUint32(header_print_size[4:], uint32(len_px))
	message_body = append(message_body, header_print_size...)
	fmt.Printf("%s", hex.Dump(message_body))
	for y := 0; y < len_px; y++ {
		content_line := make([]byte, w_bytes)
		for i := 0; i < w_bytes; i++ {
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
	get_tape_width(config)
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
	time.Sleep(100 * time.Millisecond)
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
		if i == 0x22 || i == 0x23 || i == 0x25 || i == 0x2d || i == 0x21 {
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
	index = 0x21
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
	do_print(config)
}
