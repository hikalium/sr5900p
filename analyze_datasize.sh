echo `tshark -Y 'ip.addr == 10.10.10.31' -r $1 -w - | tshark -r - -q -z follow,tcp,hex,0 | sed -E 's/^[0-9A-F]{8}  (([0-9a-f]{2} +)+).*$/\1/g' | grep -E '[0-9a-f]{2}' | xxd -r -p | wc -c` $1
