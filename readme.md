Homework 1: Gossip Messaging  
The backend webserver code is include in "main.go" code.  
The frontend is file "webpage.html".  
The gossiper code is include in "main.go".  
  
To use "main.go" code.  
I added flag -server flag that permit to run the webserver on that gossiper.  
For example:  
go run main.go -UIPort=10000 -gossipAddr=<IPAddrOfYourGossiper:PortOfYourGossiper> -name=nodeA -peers="<IPAddrOfPeer1:PortOfPeer1>,<IPAddrOfPeer2:PortOfPeer2>" -server  
If you withdraw -server flag you are by default without the backend server.  
  
To use "client/main.go" code,  
I specified one more flag -ClientPort that is the port through which the client send messages  
to the gossiper. By default it is set to 10010.  
For exameple:  
go run main.go -UIPort=10000 -msg="hello" -ClientPort=10011  
This terminal command send from client 127.0.0.1:10011 to gossiper 127.0.0.1:10000  
