#!/usr/bin/env bash

go build
cd client
go build
cd ..

BLUE='\033[0;34m'
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'
DEBUG="true"

outputFiles=()

UIPort=12345
gossipPort=5000
name='A'

private_msg="hello, this is a private message :P"
# General peerster (gossiper) command
#./Peerster -UIPort=12345 -gossipAddr=127.0.0.1:5001 -name=A -peers=127.0.0.1:5002 > A.out &

for i in `seq 1 6`;
do
	outFileName="$name.out"
	peerPort=$((($i)+5000))
	peer="127.0.0.1:$peerPort"
	gossipAddr="127.0.0.1:$gossipPort"
	./Peerster -UIPort=$UIPort -gossipAddr=$gossipAddr -name=$name -peers=$peer -rtimer=1> $outFileName &
	outputFiles+=("$outFileName")
	if [[ "$DEBUG" == "true" ]] ; then
		echo "$name running at UIPort $UIPort and gossipPort $gossipPort and peer $peer"
	fi
	UIPort=$(($UIPort+1))
	gossipPort=$(($gossipPort+1))
	name=$(echo "$name" | tr "A-Y" "B-Z")
done

sleep 4

./client/client -UIPort=12345 -msg="$private_msg" -dest=F

sleep 2
pkill -f Peerster


#testing
failed="F"

echo -e "${BLUE}###CHECK private message only displayed to F${NC}"

private_msg_received="PRIVATE origin A hop-limit 6 contents $private_msg"

if (grep -q "PRIVATE" "B.out") ; then
	failed="T"
fi
if (grep -q "PRIVATE" "C.out") ; then
	failed="T"
fi
if (grep -q "PRIVATE" "D.out") ; then
	failed="T"
fi
if (grep -q "PRIVATE" "E.out") ; then
	failed="T"
fi
if !(grep -q "$private_msg_received" "F.out") ; then
	failed="T"
fi

if [[ "$failed" == "T" ]] ; then
	echo -e "${RED}***FAILED***${NC}"
else
	echo -e "${GREEN}***PASSED***${NC}"
#	rm *.out
fi
