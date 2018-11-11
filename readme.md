The backend webserver code is in "webserver.go" code.  
The frontend and javascript is directory "./webserver_hk"
Javascript use D3 library in "./webserver_hk/lib"
The client code is in "./client/main.go"  
The gossiper code is spread in multiple files:
	"client_handler.go" handle all packets comming from client.
	"file_handler.go" handle all DataRequest and DataReply from other gossipers and FileMessage from client.
	"gossiper_utils.go" define all utils function such as prints and contain all struct definitions
	"main.go" launch the listen function on client socket of gossiper, listen function on gossip socket of gossiper and http handler if used
	"private_handler.go" handle all PrivateMessage from other gossipers and client.
	"rumor_handler.go" handle all RumorMessage from other gossipers and client.
	"simple_handler.go" handle all StatusPacket from other gossipers.
	
The script files are for testing.
  
To use "main.go" code.  
-server flag that permit to run the webserver on that gossiper.
-rtimer=<seconds> flag permit route rumoring between gossiper, no route rumoring sub routine if not set.
   
For example:  
go run main.go -UIPort=10000 -gossipAddr=<IPAddrOfYourGossiper:PortOfYourGossiper> -name=nodeA -peers="<IPAddrOfPeer1:PortOfPeer1>,<IPAddrOfPeer2:PortOfPeer2>" -server -rtimer=<Seconds>  

  
To use "client/main.go" code,  
-ClientPort flag that specify the port through which the client send messages to the gossiper. By default it is set to 10010.
-dest flag to specify to which we ask a file or to which we send a private message.
-file flag to specify the file we want to load from _SharedFiles or name of new file that will be download from another gossiper.
-msg flag to specify private or gossip message.
-request flag that specify Metahash of requested file.

If dest, file and request set together we ask for a file identified by request to a gossiper dest and we write locally the content in ./_SharedFile/file
If file set without dest and request we load a file named as specified in the flag to be able to later share it.
If dest and msg set together we send a private message to destination.
If msg set alone we send a gossip message to all peers. 

For example:  
go run main.go -UIPort=10000 -msg="hello" -ClientPort=10011 
Gossip message "hello" to all other gossipers

go run main.go -UIPort=10000 -msg="how are you?" -name=nodeA -ClientPort=10011
Send private message "how are you?" to nodeA

go run main.go -UIPort=10000 -file="text.txt" -ClientPort=10011  
Upload text.txt file that was disponible in ./_SharedFile/text.txt to make it available to other gossipers

go run main.go -UIPort=10000 -dest=nodeA -file="text2.txt" -req="715b1934521a0215b180986cdd931782a8381c58c18a9e6e7f4bfd149afd4a61" -ClientPort=10011
Download file identified by "715b1934521a0215b180986cdd931782a8381c58c18a9e6e7f4bfd149afd4a61" from nodeA and write it locally as ./_SharedFile/text2.txt 


To run use server it is recommended to launch index.html with Google Chrome