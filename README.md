# sr5900p

A command-line interface for SR5900P tape printer.

```
cargo run -- print --printer 10.10.10.31 --tcp-data sample_tcp_data/w18_hikalium.bin
```

## How to lookup the printer's IP
```
dns-sd -L "KING JIM TEPRA PRO SR5900P" _pdl-datastream._tcp local | grep -o -E 'SR5900[0-9A-Za-z]+.local'

# will give you "SR5900PA28A76.local" or something. With the domain name, you can lookup the address like this:

dns-sd -Gv4v6 SR5900PA28A76.local

# DATE: ---Mon 28 Nov 2022---
#  5:00:25.259  ...STARTING...
# Timestamp     A/R    Flags if Hostname                               Address                                      TTL
#  5:00:25.260  Add 40000002  7 SR5900PA28A76.local.                   10.10.10.31                                  120
```

## How to extract TCP data
```
# update these values to match with your env
export IFACE=en0
export DUMP_LABEL=w18_Aaa
export DEVICE_IP=10.10.10.31

# take dump
sudo tcpdump -i ${IFACE} -w ${DUMP_LABEL}.pcapng
# print via the GUI, then stop the capture with Ctrl-C

# extract tcp stream from the dump, data part only
tshark -Y "ip.addr == ${DEVICE_IP}" -r ${DUMP_LABEL}.pcapng -w - | \
tshark -r - -q -z follow,tcp,hex,0 | \
sed -E 's/^[0-9A-F]{8}  (([0-9a-f]{2} +)+).*$/\1/g' | \
grep -E '[0-9a-f]{2}' | xxd -r -p | dd status=none bs=1 skip=14 > ${DUMP_LABEL}.bin

# and have fun!
cargo run -- analyze --tcp-data ${DUMP_LABEL}.bin
```

## License
MIT

## Author
hikalium

## Special Thanks
Mine02C4 (for [the initial analysis of the protocol](https://github.com/Mine02C4/TEPRA_PRO_SR5900P_analysis) )
