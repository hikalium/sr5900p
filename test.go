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
		0x54, 0x50, 0x52, 0x54,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x01,
		0x00, 0x00, 0x00, 0x20,
		0x00, 0x00, 0x00, 0x01, // cmd(1)
		0x00, 0x00, 0x00, 0x00, // data size (0)
		0x0a, 0x0a, 0x0a, 0x5a, // ipv4addr
		0x00, 0x00, 0x00, 0x00,
	},
	/*
		74 70 72 74 // "tprt"
		00 00 00 00 // const
		00 00 00 01 // const
		00 00 00 20 // const
		00 00 00 01 // cmd (1)
		00 00 00 14 // data size (0x14)
		00 00 00 00 // ip addr
		00 00 00 00 // token?
		// data
		// 14 00 00 05
		// 00 00 00 00
		// 40 00 00 00
		// 00 XX 00 00 // XX@+0x2d 00: printing, 01: done?
		// 00 00 00 00
	*/

	"print_start": {
		// cmd == 0x02
		0x54, 0x50, 0x52, 0x54,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x01,
		0x00, 0x00, 0x00, 0x20,
		0x00, 0x00, 0x00, 0x02, // cmd (2)
		0x00, 0x00, 0x00, 0x00, // data size (0)
		0x0a, 0x0a, 0x0a, 0x5a, // ipv4addr
		0x00, 0x00, 0x00, 0x00,
	},
	/*
		74 70 72 74
		00 00 00 00
		00 00 00 01
		00 00 00 20
		00 00 00 02 cmd (2)
		00 00 00 03 data size (0)
		00 00 00 00
		00 00 00 00
		02 00 00
	*/

	"print_stop": {
		0x54, 0x50, 0x52, 0x54,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x01,
		0x00, 0x00, 0x00, 0x20,
		0x00, 0x00, 0x00, 0x03, // cmd == 3
		0x00, 0x00, 0x00, 0x00, // data size == 0
		0x0a, 0x0a, 0x0a, 0x5a, // ipv4addr
		0x00, 0x00, 0x00, 0x00,
	},
	/*
		74 70 72 74
		00 00 00 00
		00 00 00 01
		00 00 00 20
		00 00 00 03 // cmd == 3
		00 00 00 03 // data_size == 3
		00 00 00 00
		00 00 00 00
		// data
		03 00 00                                          |...|
	*/

	"request04": {
		// cmd == 0x04
		0x54, 0x50, 0x52, 0x54, // "TPRT"
		0x00, 0x00, 0x00, 0x00, // const
		0x00, 0x00, 0x00, 0x01, // const
		0x00, 0x00, 0x00, 0x20, // const
		0x00, 0x00, 0x00, 0x04, // cmd (4)
		0x00, 0x00, 0x00, 0x00, // data size (0)
		0x0a, 0x0a, 0x0a, 0x5a, // ipv4addr
		0x00, 0x00, 0x00, 0x00,
	},
	/*
		74 70 72 74 // "tprt"
		00 00 00 00 // const
		00 00 00 01 // const
		00 00 00 20 // const
		00 00 00 04 // const
		00 00 00 03 // data size (3)
		00 00 00 00 // ip addr
		00 00 00 00 // ???? (same random value as request)
		04 01 00    // ??? (e.g. 040100(idle?), 040200 (busy?))
	*/

	"requestName": {
		// cmd == 0x05
		0x54, 0x50, 0x52, 0x54, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x20,
		0x00, 0x00, 0x00, 0x05, 0x00, 0x00, 0x00, 0x00,
		0x0a, 0x0a, 0x0a, 0x5a, // ipv4addr
		0x00, 0x00, 0x00, 0x00,
	},
	// 00000000  74 70 72 74 00 00 00 00  00 00 00 01 00 00 00 20  |tprt........... |
	// 00000010  00 00 00 05 00 00 00 40  00 00 00 00 00 00 00 00  |.......@........|
	// 00000020  54 45 50 52 41 20 50 52  4f 20 53 52 35 39 30 30  |TEPRA PRO SR5900|
	// 00000030  50 00 00 00 00 00 00 00  00 00 00 00 00 00 00 00  |P...............|
	// 00000040  53 52 35 39 30 30 50 41  32 38 41 37 36 00 00 00  |SR5900PA28A76...|
	// 00000050  00 00 00 00 00 00 00 00  00 00 00 00 00 00 00 00  |................|
	"request100": {
		0x54, 0x50, 0x52, 0x54, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x20,
		0x00, 0x00, 0x01, 0x00, // cmd (0x0100)
		0x00, 0x00, 0x00, 0x00,
		0x0a, 0x0a, 0x0a, 0x5a, // ipv4addr
		0x00, 0x00, 0x00, 0x00,
	},
	"request101": {
		0x54, 0x50, 0x52, 0x54, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x20,
		0x00, 0x00, 0x01, 0x01, // cmd (0x0100)
		0x00, 0x00, 0x00, 0x00,
		0x0a, 0x0a, 0x0a, 0x5a, // ipv4addr
		0x00, 0x00, 0x00, 0x00,
	},
	// ================================================ [IP    ] [  job?]
	// 545052540000000000000001000000200000010000000000 0a0a0a5a a36bc80f
	// 747072740000000000000001000000200000010000000001 0a0a0a5a a36bc80f 10
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

var data_tape_feed_and_cut = []byte{
	0x1b, 0x7b, 0x04, 0x2b, 0x01, 0x2c, 0x7d,
}
var data_tape_feed = []byte{
	0x1b, 0x7b, 0x04, 0x2b, 0x00, 0x2b, 0x7d,
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

var header_print_size = []byte{
	0x1b, 0x7b, 0x07, 0x4c, 0x4c,
	0xa9, 0x01, 0x00, 0x00, 0xf6, 0x7d, 0x1b, 0x7b, 0x05, 0x54, 0x91, 0x00, 0xe5, 0x7d, // 60+
	//0xa9, 0x01, 0x00, 0x00, 0xf6, 0x7d, 0x1b, 0x7b, 0x05, 0x54, 0x00, 0x00, 0x00, 0x7d, // 60+
	//0x35, 0x02, 0x00, 0x00, 0x83, 0x7d, 0x1b, 0x7b, 0x05, 0x54, 0x00, 0x00, 0x00, 0x7d, // 60+
	//0x6b, 0x03, 0x00, 0x00, 0xba, 0x7d, 0x1b, 0x7b, 0x05, 0x54, 0x96, 0x00, 0xea, 0x7d,
}

/*
1b 7b 07 4c 4c __ __ 00 00 __ 7d 1b 7b 05 54 __ 00 __ 7d common
           L = a9 01       f6 = 30mm
           L = 35 02       83 = 40mm
           L = c1 02       0f = 50mm
           L = 07 03       56 = 55mm
           L = 52 03       a1 = 60mm
           L = 5c 03       ab = 61mm
           L = 6b 03       ba = 62mm
           L = 87 05       d8 = 100mm
           L = 13 06       65 = 110mm
           L = a4 06       f6 = 120mm
                                             91    e5    ./dump_W12mm_L110mm.bin
                                             91    e5    ./dump_W12mm_L50mm.bin
                                             91    e5    // W=12mm L=50mm

                                             92    e6    ./dump_W12mm_L100mm.bin
                                             92    e6    ./dump_W12mm_L40mm.bin
                                             92    e6    // W=12mm L=40mm

                                             93    e7    ./dump_W12mm_L120mm.bin
                                             93    e7    ./dump_W12mm_L30mm.bin
                                             93    e7    // W=12mm L=30mm

                                             94    e8    ./dump_W18mm_L50mm.bin
                                             94    e8    // W=18mm L=50mm

                                             95    e9    ./dump_W18mm_L40mm.bin
                                             95    e9    // W=18mm L=40mm

                                             96    ea    ./dump_W18mm_L30mm.bin
                                             96    ea    ./dump_W18mm_L30mm_2.bin
                                             96    ea    ./dump_W24mm_L110mm.bin
                                             96    ea    ./dump_W24mm_L110mm_2.bin
                                             96    ea    ./dump_a_T24mm_W24mm_L62mm.bin
                                             96    ea    ./dump_ab_T24mm_W24mm_L50mm.bin
                                             96    ea    ./dump_ab_T24mm_W24mm_L55mm.bin
                                             96    ea    ./dump_ab_T24mm_W24mm_L61mm.bin
                                             96    ea    ./dump_abc_T24mm_W24mm_L61mm.bin
                                             96    ea    ./dump_abc_T24mm_W24mm_L62mm.bin
                                             96    ea    // W=18mm L=30mm

                                             98    ec    ./dump_ab_T24mm_W24mm_L60mm.bin
                                             XX    YY    XX + 0x54 == YY
*/

// 360dpi =>
// px = mm/25.4*360
// mm = px/360*25.4

// 384dot =>
// 48*8 bits => 48 bytes per row is the max

var header_per_line = []byte{
	// 1b2e00000001 [width_in_px: u16]
	0x1b, 0x2e, 0x00, 0x00, 0x00, 0x01, 0x1d, 0x01, // 285px == ~20mm, for 24mm tape
	// 18mm : 1b2e 0000 0001 d700
	// 12mm : 1b2e 000a 0a0a 9000 << ???
}

var termination = []byte{
	// constant
	0x0c,
	0x1b, 0x7b, 0x03, 0x40, 0x40, 0x7d,
}

func doPrint(config *TestConfig) error {
	w_px := 384
	w_bytes := (w_px + 7) / 8
	log.Printf("w_bytes = %v", w_bytes)
	len_mm := 40.0
	len_px := int(len_mm * 360.0 / 25.4)
	log.Printf("len_px = 0x%08x", len_px)
	message_header := header_common
	//binary.LittleEndian.PutUint32(header_print_size[4:], uint32(len_px))
	message_header = append(message_header, header_print_size...)
	message_body := make([]byte, 0)
	fmt.Printf("%s", hex.Dump(message_body))
	for y := 0; y < len_px; y++ {
		content_line := make([]byte, w_bytes)
		for i := 0; i < w_bytes; i++ {
			chunk := 0x01
			if y%8 == 0 {
				chunk = 0xff
			}
			for k := 0; k < 8; k++ {
				x := i*8 + (7 - k)
				if x == y%w_px {
					chunk = chunk | (1 << k)
				}
			}
			content_line[i] = byte(chunk)
		}
		line := append(header_per_line, content_line...)
		message_body = append(message_body, line...)
	}
	message_body = append(message_body, termination...)

	// 1 > 1 > 4 > 2 > tcp_open > 0x101 > 0x100 > 1 > data_header > 1 > data > 1 > data > ...
	/*
		for is_printing(config) {
			log.Print("Aborting previous print job...\n")
			testUDPMessage(config, "print_stop")
			time.Sleep(500 * time.Millisecond)
		}
	*/

	get_tape_width(config)
	get_tape_width(config)
	testUDPMessage(config, "request04")
	testUDPMessage(config, "print_start")

	conn, err := net.Dial("tcp4", config.RemoteIpAddr+":9100")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	testUDPMessage(config, "request101")
	testUDPMessage(config, "request100")

	_, err = conn.Write(message_header)
	if err != nil {
		log.Fatal(err)
	}
	get_tape_width(config)
	_, err = conn.Write(message_body)
	if err != nil {
		log.Fatal(err)
	}
	for {
		time.Sleep(500 * time.Millisecond)
		if is_printing(config) {
			log.Print("waiting...\n")
			continue
		}
		break
	}
	time.Sleep(100 * time.Millisecond)
	testUDPMessage(config, "print_stop")
	return nil
}

func doFeedAndCut(config *TestConfig) error {
	// 1 > 2 > tcp_open > 0x101 > 0x100 > 1 > data_header > 1 > data > 1 > data > ...
	/*
		for is_printing(config) {
			log.Print("Aborting previous print job...\n")
			testUDPMessage(config, "print_stop")
			time.Sleep(500 * time.Millisecond)
		}
	*/

	get_tape_width(config)
	testUDPMessage(config, "print_start")

	conn, err := net.Dial("tcp4", config.RemoteIpAddr+":9100")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	testUDPMessage(config, "request101")
	testUDPMessage(config, "request100")
	_, err = conn.Write(data_tape_feed_and_cut)
	if err != nil {
		log.Fatal(err)
	}
	for {
		time.Sleep(500 * time.Millisecond)
		if !is_feeding(config) {
			log.Print("waiting0...\n")
			continue
		}
		break
	}
	log.Print("feed started!")
	for {
		time.Sleep(500 * time.Millisecond)
		if is_feeding(config) {
			log.Print("waiting1...\n")
			continue
		}
		break
	}
	time.Sleep(100 * time.Millisecond)
	testUDPMessage(config, "print_stop")
	return nil
}

func is_printing(config *TestConfig) bool {
	res := testUDPMessage(config, "get_printer_status")
	v := res[0x2d]
	if v != 0 && v != 1 {
		log.Fatalf("Unexpected printing status %02x\n", v)
	}
	return v == 0
}

func is_feeding(config *TestConfig) bool {
	res := testUDPMessage(config, "get_printer_status")
	v := res[0x21]
	if v != 0 && v != 1 {
		log.Fatalf("Unexpected printing status %02x\n", v)
	}
	return v == 1
}

func get_tape_width(config *TestConfig) {
	res := testUDPMessage(config, "get_printer_status")
	expected := []byte{
		0x74, 0x70, 0x72, 0x74,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x01,
		0x00, 0x00, 0x00, 0x20,
		0x00, 0x00, 0x00, 0x01,
		0x00, 0x00, 0x00, 0x14,
		0x0a, 0x0a, 0x0a, 0x5a,
		0x00, 0x00, 0x00, 0x00,
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
	doPrint(config)
	//doFeedAndCut(config)
}
