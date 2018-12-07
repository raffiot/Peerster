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

# General peerster (gossiper) command
#./Peerster -UIPort=12345 -gossipAddr=127.0.0.1:5001 -name=A -peers=127.0.0.1:5002 > A.out &

for i in `seq 1 6`;
do
	outFileName="$name.out"
	peerPort=$(((i)%6+5000))
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

./client/client -UIPort=12345 -file=test.txt
#"715b1934521a0215b180986cdd931782a8381c58c18a9e6e7f4bfd149afd4a61"
./client/client -UIPort=12345 -file=rimbaud.txt
sleep 5

pkill -f Peerster


failed="T"
echo -e "${RED}###CHECK FOUND-BLOCK IN AT LEAST ONE OF THE OUTPUT FILE${NC}"
for i in `seq 0 5`;
do
	echo -e "${outputFiles[$i]}"
    if (grep -q "FOUND-BLOCK" "${outputFiles[$i]}") ; then
        failed="F"
    fi
done

if [[ "$failed" == "T" ]] ; then
	echo -e "${RED}***FAILED***${NC}"
else
	echo -e "${GREEN}***PASSED***${NC}"
fi

failed="F"
echo -e "${RED}###CHECK RIMBAUD IN ALL CHAINS${NC}"
for i in `seq 0 5`;
do
	
    if !(grep -q "rimbaud.txt" "${outputFiles[$i]}") ; then
	echo -e "${outputFiles[$i]}"
        failed="T"
    fi
done

if [[ "$failed" == "T" ]] ; then
	echo -e "${RED}***FAILED***${NC}"
else
	echo -e "${GREEN}***PASSED***${NC}"
fi
