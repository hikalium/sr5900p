#!/bin/bash
ifconfig | grep eth | cut -d ' ' -f 2 | sort | uniq | tee mac_block_list.txt
while true
do
	NEW_MAC_LIST=`ifconfig | grep eth | cut -d ' ' -f 2 | sort | uniq | grep -v -F -f mac_block_list.txt`
	RETCODE=$?
	if [ ${RETCODE} -ne 0 ]
	then
		echo "not found"
	else
		echo "found"
		echo "${NEW_MAC_LIST}" | xargs -I {} -- bash -c 'cargo run -- print --printer 10.10.10.44 --mac-addr {} && echo {} >> mac_block_list.txt'
	fi
	sleep 1
done
