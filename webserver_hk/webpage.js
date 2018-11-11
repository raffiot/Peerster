var private_msg = false;
var private_id = "";


function send(type, msg) {
	var d;
	if(type === "private"){
		d = {message:msg,destination:private_id}
	}else{
		d = {value:msg}
	}

  if (msg.length > 0) {
    $.ajax({
      type: "POST",
      url: "http://localhost:8080/"+type,
      data: d
    })
    .done(function(data) {
      if (!(type === "node") ){
        document.getElementById("message").value = '';
      }
    });
  }
  get_message_and_nodes()
}



function appendPublic(){
	var div = document.createElement("div");
	div.setAttribute('class','peer_group');
	
	var b = document.createElement("b");
	b.id = "public";
	b.onclick = privateDiscussion;
	b.style.cursor =  "pointer";
	b.style.margin =  "auto";
	b.appendChild(document.createTextNode("public"));
	div.appendChild(b)

	

	var button = document.createElement("button");
	button.setAttribute('class','btn btn-default');
	button.type = "submit";
	button.id = "upload_public";
	button.onclick=fileup
	button.style.float = "right";
	var span = document.createElement("span");
	span.setAttribute('class', 'glyphicon glyphicon-upload');
	span.setAttribute('aria-hidden','true');
	button.appendChild(span);

	div.appendChild(button);
	
	document.querySelector("#discussions-box").appendChild(div);

}

function appendPeer(s){
	
	var div = document.createElement("div");
	div.setAttribute('class','peer_group');
	div.id = "private_peer"
	var b = document.createElement("b");
	b.id = s;
	b.onclick = privateDiscussion;
	b.style.cursor =  "pointer";
	b.style.margin =  "auto";
	b.appendChild(document.createTextNode(s));
	div.appendChild(b)

	

	var button = document.createElement("button");
	button.setAttribute('class','btn btn-default');
	button.type = "submit";
	button.id = s;
	button.onclick=fileRequest;
	button.style.float = "right";
	var span = document.createElement("span");
	span.setAttribute('class', 'glyphicon glyphicon-file');
	span.setAttribute('aria-hidden','true');
	button.appendChild(span);

	div.appendChild(button);
	
	document.querySelector("#discussions-box").appendChild(div);

}




$(document).ready(function(){
	appendPublic();
  $("button#appendPeer").click(function(e){
    var new_peer = prompt("Please enter new peer:", "IP_Address:Port");
	if (!(new_peer == null || new_peer == "")) {
		send("node",new_peer);
	}
  });

  $("#message").keydown(function(e) {
    if (e.keyCode == 13) {
			var txt = document.getElementById("message").value;
			if(private_msg){
				send("private",txt)
			} else {
				send("message",txt);
			}
    }
  });

  $("button#send").click(function(e){
		var txt = document.getElementById("message").value;
		if(private_msg){
			send("private",txt);
		} else {
			send("message",txt);
		}
  });
});

$.ajax({
  type: "GET",
  url: "http://localhost:8080/id",
  success: function(data,status,xhr){
    var p = document.createElement("p");
    var dataJSON = JSON.parse(data);
    document.getElementById("title").innerHTML = "Peerster " + dataJSON.toString();
  }
});

function privateDiscussion(event){
	//console.log("in privateDiscussion")
	var ID = event.target.id;
	//console.log(ID)
	document.getElementById("msg-box").innerHTML = '';
	if(ID ==="public"){
		private_msg = false;
		private_id = "";
		document.getElementById("message").placeholder = "Write message here...";
		
	} else{
		
		private_msg = true;
		private_id = ID;
		document.getElementById("message").placeholder = "Write private message to "+ID+" here...";
	}
	
	var childDivs = document.getElementById("discussions-box").getElementsByTagName('div');
	for( i=0; i< childDivs.length; i++ ){
		childDivs[i].style.background = "#ffffff";
	}
	event.target.parentElement.style.background = "#85c1e9";
	get_message_and_nodes();
}


function fileRequest(event){
	var ID = event.target.id;
	var req = prompt("Please enter the Name and Methash of the file you want from "+ID+":", "Filename:Metahash");
	if (!(req == null || req == "")) {
		var res = req.split(":")
		var file = res[0]
		var metahash = res[1]
		if ((file != "") && (metahash != "")){
			$.ajax({
			  type: "POST",
			  url: "http://localhost:8080/file",
			  data: {type:"request",filename:file,destination:ID,request:metahash}
			});
		}
	}
	
}


function fileup(){

	var req = prompt("Please enter the name of the file you want to share:", "filename");
	if (!(req == null || req == "")) {
		$.ajax({
		  type: "POST",
		  url: "http://localhost:8080/file",
		  data: {type:"upload",filename:req}
		});
	}
	
}

setInterval(get_message_and_nodes, 1000);

function get_message_and_nodes(){
	if(!private_msg){
	  $.ajax({
		type: "GET",
		url: "http://localhost:8080/message",
		success: function(data,status,xhr){
		  var out = document.querySelector("#msg-box");

		  const isScrolledToBottom = out.scrollHeight - out.clientHeight <= out.scrollTop + 1

		  out.innerHTML = "";
		  var dataJSON = JSON.parse(data);
		  for (x in dataJSON) {
			var p = document.createElement("p");
			var msg = dataJSON[x.toString()];

			var b = document.createElement("b");
			b.appendChild(document.createTextNode(msg.Origin + " " + msg.ID  + " : "))

			var text = document.createTextNode(msg.Text);
			p.appendChild(b);
			p.appendChild(text);
			document.querySelector("#msg-box").appendChild(p);
		  }

		  // scroll to bottom if isScrolledToBottom is true
		  if (isScrolledToBottom) {
			out.scrollTop = out.scrollHeight - out.clientHeight;
		  }

		}
	  });
	} else {
		$.ajax({
			type: "GET",
			url: "http://localhost:8080/private",
			data: private_id,
			success: function(data,status,xhr){
				var out = document.querySelector("#msg-box");

				const isScrolledToBottom = out.scrollHeight - out.clientHeight <= out.scrollTop + 1

				out.innerHTML = "";
				var dataJSON = JSON.parse(data);
				for (x in dataJSON) {
					var p = document.createElement("p");
					var msg = dataJSON[x.toString()];

					var b = document.createElement("b");
					b.appendChild(document.createTextNode(msg.Origin + " : "))

					var text = document.createTextNode(msg.Text);
					p.appendChild(b);
					p.appendChild(text);
					document.querySelector("#msg-box").appendChild(p);
				}

				// scroll to bottom if isScrolledToBottom is true
				if (isScrolledToBottom) {
					out.scrollTop = out.scrollHeight - out.clientHeight;
				}
			}
		});
	}

  $.ajax({
    type: "GET",
    url: "http://localhost:8080/node",
    success: function(data,status,xhr){
      document.querySelector("#peers-box").innerHTML = ""

      var dataJSON = JSON.parse(data);
      for(x in dataJSON){
		var p = document.createElement("h5");
        var msg = dataJSON[x.toString()];
        var text = document.createTextNode(x.toString());
        p.appendChild(text);
        document.querySelector("#peers-box").appendChild(p);
      }
    }
  });

  $.ajax({
    type: "GET",
    url: "http://localhost:8080/peer",
    success: function(data,status,xhr){
		var dataJSON = JSON.parse(data)
		//console.log(dataJSON)
		var la = d3.selectAll("#discussions-box")
		la.selectAll("#private_peer")
			.data(dataJSON)
			.enter()
			.each(d => appendPeer(d.toString()));
			//.append("b")
			//.text(d => d.toString());

		/**
    document.querySelector("#discussions-box").innerHTML = "";
	appendPublic()
	var dataJSON = JSON.parse(data)
	for(x in dataJSON){
		appendPeer(dataJSON[x.toString()]);
	}*/
	/**	
      var dataJSON = JSON.parse(data);
	  len = Object.keys(dataJSON).length;
	  if ( len * 2 +1 > document.querySelector("#discussions-box").childNodes.length){
		  for(x in dataJSON){
			if (x == len-1){
				appendPeer(dataJSON[x.toString()]);
			}
		}
	  }*/
	  
    }
  });
}

function previewFile() {
	//MAYBE WE DON'T WANT THIS
  var file    = document.querySelector('input[type=file]').files[0];
  var reader  = new FileReader();

  reader.onloadend = function () {
    console.log(reader.result);
  }

  if (file) {
    reader.readAsDataURL(file);
  }
}
