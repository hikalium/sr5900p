analyze:
	cargo run -- analyze --tcp-data sample_tcp_data/w18_Aaa.bin

print:
	cargo run -- print --printer 10.10.10.31 --tcp-data sample_tcp_data/w18_hikalium.bin

analyze_all:
	find ./sample_tcp_data/*.bin | xargs -I {} -- bash -c 'echo "*** {}" && cargo run -q -- analyze --tcp-data {} | grep -v "cmd 0x1b 0x2e"'
