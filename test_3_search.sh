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
./client/client -UIPort=12345 -file=espana.txt
./client/client -UIPort=12347 -file=rimbaud.txt
sleep 4

./client/client -UIPort=12346 -keywords=es -budget=3

sleep 5

./client/client -UIPort=12345 -keywords=rim -budget=0

sleep 5

./client/client -UIPort=12346 -file=new_test.txt -request=715b1934521a0215b180986cdd931782a8381c58c18a9e6e7f4bfd149afd4a61

sleep 5
pkill -f Peerster

failed="F"
echo -e "${RED}###CHECK B FIND MATCH FROM A for test.txt${NC}"
if !(grep -q "FOUND match test.txt at " "B.out") ; then
	failed="T"
fi
if !(grep -q "FOUND match espana.txt at " "B.out") ; then
	failed="T"
fi
if !(grep -q "SEARCH FINISHED" "B.out") ; then
	failed="T"
fi

if [[ "$failed" == "T" ]] ; then
	echo -e "${RED}***FAILED***${NC}"
else
	echo -e "${GREEN}***PASSED***${NC}"
fi

failed="F"
echo -e "${RED}###CHECK A FIND MATCH FROM C for rimbaud.txt${NC}"
if !(grep -q "FOUND match rimbaud.txt at C" "A.out") ; then
	failed="T"
fi

if [[ "$failed" == "T" ]] ; then
	echo -e "${RED}***FAILED***${NC}"
else
	echo -e "${GREEN}***PASSED***${NC}"
fi

echo -e "${RED}###CHECK B DOWNLOAD test.txt from A${NC}"
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

failed="F"
echo -e "${RED}###CHECK NO CHUNK WITH INDEX 0${NC}"
for i in `seq 0 5`;
do
	echo -e "${outputFiles[$i]}"
    if (grep -q "chunks=0" "${outputFiles[$i]}") ; then
        failed="T"
    fi
done

if [[ "$failed" == "T" ]] ; then
	echo -e "${RED}***FAILED***${NC}"
else
	echo -e "${GREEN}***PASSED***${NC}"
fi


