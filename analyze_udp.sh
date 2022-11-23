echo $1
tshark -Y 'ip.addr == 10.10.10.31 && udp' -r $1 -T fields -e data | sed -E 's/^(.{36})(.{4})(.{8})(.{8})(.{4})(.{4})(.*$)/\1 \2 \3 \4 \5 \6 \7/'| sort | uniq
