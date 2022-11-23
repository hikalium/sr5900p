package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"net"
)

type TestConfig struct {
	RemoteIpAddr string `json:"remoteIpAddr"`
}

func getConfig() (*TestConfig, error) {
	/*
		raw, err := ioutil.ReadFile("test.config.json")
		if err != nil {
			return nil, err
		}
		config := &TestConfig{}
		err = json.Unmarshal(raw, config)
		if err != nil {
			return nil, err
		}
	*/
	return &TestConfig{
		RemoteIpAddr: "10.10.10.31",
	}, nil
}

var requestDefinitions = map[string][]byte{
	"get_tape_info": {
		0x54, 0x50, 0x52, 0x54, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x20,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	},
	// 00000000  74 70 72 74 00 00 00 00  00 00 00 01 00 00 00 20  |tprt........... |
	// 00000010  00 00 00 01 00 00 00 14  00 00 00 00 00 00 00 00  |................|
	// 00000020  14 00 00 05 00 00 00 00  40 00 00 00 00 01 00 00  |........@.......|
	// 00000030  00 00 00 00                                       |....|

	"print_start": {
		0x54, 0x50, 0x52, 0x54, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x20,
		0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	},
	// 00000000  74 70 72 74 00 00 00 00  00 00 00 01 00 00 00 20  |tprt........... |
	// 00000010  00 00 00 02 00 00 00 03  00 00 00 00 00 00 00 00  |................|
	// 00000020  02 00 00

	"request03": {
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
	fmt.Printf("%s", hex.Dump(buffer[:length]))
	//fmt.Println(hex.EncodeToString(buffer[:length]))
	return buffer[:length]
}

// [header (66 bytes)] ([separator (10 bytes)][data(34 bytes)])*435(0x1b3) [terminator(7 bytes)]
// 19213 = 66 + 44 * 435 + 7

// header
// 1b7b 07 7b00005354227d
// 1b7b 07 4302020101497d
// 1b7b 04 4405497d
// 1b7b 03 47477d
// 1b7b 04 7300737d
// 1b7b 05 6c0505767d
// 1b7b 07 4cb3010000007d
// 1b7b 05 549b00ef7d

// separator (10 bytes)
// 1b 2e 00 00 00 01 1d 01 00 00
// 34 bytes
// 44 bytes for one line (height) when 24mm tape
// 2 byte per 1mm?

// terminator (7 bytes)
//		0c 1b 7b 03 40407d

var header = []byte{
	0x1b, 0x7b, 0x07, 0x43, 0x02, 0x02, 0x01, 0x01, 0x49, 0x7d,
	0x1b, 0x7b, 0x04, 0x44, 0x05, 0x49, 0x7d,
	0x1b, 0x7b, 0x05, 0x6c, 0x05, 0x05, 0x76, 0x7d,
	0x1b, 0x7b, 0x04, 0x73, 0x00, 0x73, 0x7d,
	0x1b, 0x7b, 0x03, 0x47, 0x47, 0x7d,
	0x1b, 0x7b, 0x07, 0x4c, 0xe2, 0x00, 0x00, 0x00, 0x2e, 0x7d, // [4..8]: len?
	0x1b, 0x7b, 0x05, 0x54, 0x40, 0x00, 0x94, 0x7d,
}

var header_per_line = []byte{
	0x1b, 0x2e, 0x00, 0x0a, 0x0a, 0x01, 0x90, 0x00,
}

func testPrint(config *TestConfig) error {
	if len(header) != 56 {
		return fmt.Errorf("Header length MUST be 56 Bytes")
	}
	message_body := header
	for y := 0; y < 0xe2; y++ {
		content_line := make([]byte, 34)
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
	termination := []byte{
		0x0c, 0x1b, 0x7b, 0x03, 0x40, 0x40, 0x7d,
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
}

func get_tape_width(config *TestConfig) {
	res := testUDPMessage(config, "get_tape_info")
	expected := []byte{
		0x74, 0x70, 0x72, 0x74, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x20,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x14, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x14, 0x00, 0x06, 0x00, 0x00, 0x00, 0x00, 0x00, 0x40, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
	}
	fmt.Printf("%s", hex.Dump(res))
	for i := 0; i < len(expected); i++ {
		if i == 0x22 || i == 0x23 || i == 0x25 {
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
	//do_print(config)
	get_tape_width(config)
}
