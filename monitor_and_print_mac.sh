#!/bin/bash
function expect_arg() {
	VALUE="$1"
	NAME="$2"
	if [ -z $1 ]; then
		echo "Expected arg ${NAME}";
		exit 1;
	else
		export "${NAME}"="${VALUE}"
		echo "${NAME}"="${VALUE}"
	fi
}
expect_arg "$1" "PRINTER_IP"
function list_mac {
	ifconfig | grep ether | sed -E 's/^.*ether ([a-f0-9\:]+).*$/\1/' | sort | uniq
}
list_mac | tee mac_block_list.txt
while true
do
	NEW_MAC_LIST=`ifconfig | grep ether | sed -E 's/^.*ether ([a-f0-9\:]+).*$/\1/' | sort | uniq | grep -v -F -f mac_block_list.txt`
	RETCODE=$?
	if [ ${RETCODE} -ne 0 ]
	then
		echo "not found"
	else
		echo "found"
		echo "${NEW_MAC_LIST}" | xargs -I {} -- bash -c "./sr5900p print --printer ${PRINTER_IP} --mac-addr {} && echo {} >> mac_block_list.txt"
	fi
	sleep 1
done
