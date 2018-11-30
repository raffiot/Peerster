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

./client/client -UIPort=12345 -file=test.txt

sleep 4

./client/client -UIPort=12350 -file="new_test.txt" -dest=A -request="715b1934521a0215b180986cdd931782a8381c58c18a9e6e7f4bfd149afd4a61"

sleep 15

pkill -f Peerster

diff_res=$(diff _SharedFiles/test.txt _Downloads/new_test.txt)

if [[ ! -f ./_Downloads/new_test.txt ]]; then
	echo -e "${RED}***FAILED***${NC}"
else
    if [[ "$diff_res" != "" ]]; then
        echo -e "${RED}***FAILED***${NC}"
    else
        echo -e "${GREEN}***PASSED***${NC}"
    fi

#    rm _Downloads/hamlet_F.txt
#    rm _Downloads/.meta/hamlet_F.txt
fi
