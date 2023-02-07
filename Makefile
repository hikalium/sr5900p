build:
	cargo build --release

test:
	cargo run --release -- print --test-pattern --dry-run
	diff preview.png assets/test_pattern.png && echo OK || { echo NG ; exit 1 ; }

run:
ifndef PRINTER_IP
	$(error Please set PRINTER_IP)
endif
	cargo run -- print --printer ${PRINTER_IP} --test-pattern

build_static:
	RUSTFLAGS='-C target-feature=+crt-static' cargo build --target x86_64-unknown-linux-gnu

install:
	cargo install --path .

analyze:
	cargo run -- analyze --tcp-data sample_tcp_data/w18_Aaa.bin

print:
	cargo run -- print --printer 10.10.10.31 --tcp-data sample_tcp_data/w18_hikalium.bin

analyze_all:
	find ./sample_tcp_data/*.bin | xargs -I {} -- bash -c 'echo "*** {}" && cargo run -q -- analyze --tcp-data {} | grep -v "cmd 0x1b 0x2e"'

gen:
	cargo run -- print --printer 10.10.10.31 --gen-test --dry-run

gen_print:
	cargo run -- print --printer 10.10.10.31 --gen-test
