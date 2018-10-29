#!/usr/bin/env bash

go build
cd client
go build
cd ..

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'
DEBUG="true"

outputFiles=()
message_c1=Free_kanel_bulle
message_c2=I_m_comming_hungry
message_c3=Where_is_Zidjie
message_c4=kom_pa_711
message_c5=ur_ma_du_filippa

./Peerster -UIPort=10000 -gossipAddr=127.0.0.1:5000 -name=tibo -peers="127.0.0.1:5001" -rtimer=3 > tibo.out &
./Peerster -UIPort=10001 -gossipAddr=127.0.0.1:5001 -name=valou -peers="127.0.0.1:5002" -rtimer=3 > valou.out &
./Peerster -UIPort=10002 -gossipAddr=127.0.0.1:5002 -name=marc -peers="127.0.0.1:5003" -rtimer=3 > marc.out &
./Peerster -UIPort=10003 -gossipAddr=127.0.0.1:5003 -name=lionel -peers="127.0.0.1:5004" -rtimer=3 > lionel.out &
./Peerster -UIPort=10004 -gossipAddr=127.0.0.1:5004 -name=anton -peers="" -rtimer=3 > anton.out &
sleep 2

./client/client -UIPort=10004 -msg=$message_c1

sleep 10
pkill -f Peerster

failed="F"

echo -e "${RED}###CHECK chain end receive message${NC}"

if !(grep -q "RUMOR origin anton from 127.0.0.1:5001 ID 1 contents $message_c1" "tibo.out") ; then
	failed="T"
fi

if [[ "$failed" == "T" ]] ; then
	echo -e "${RED}***FAILED***${NC}"
else
	echo -e "${GREEN}***PASSED***${NC}"
fi

failed="F"

echo -e "${RED}###CHECK DSDV${NC}"

if !(grep -q "DSDV anton 127.0.0.1:5001" "tibo.out") ; then
	failed="T"
fi
if !(grep -q "DSDV anton 127.0.0.1:5002" "valou.out") ; then
	failed="T"
fi
if !(grep -q "DSDV anton 127.0.0.1:5003" "marc.out") ; then
	failed="T"
fi
if !(grep -q "DSDV anton 127.0.0.1:5004" "lionel.out") ; then
	failed="T"
fi

if [[ "$failed" == "T" ]] ; then
	echo -e "${RED}***FAILED***${NC}"
else
	echo -e "${GREEN}***PASSED***${NC}"
fi
