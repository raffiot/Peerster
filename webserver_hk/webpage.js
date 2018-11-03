var private_msg = false;
var private_id = "";

		
function send(type, message) {
  

  if (message.length > 0) {
    $.ajax({
      type: "POST",
      url: "http://localhost:8080/"+type,
      data: {value:message}
    })
    .done(function(data) {
      if (type === "message"){
        document.getElementById("message").value = '';
      }
    });
  }
  
  get_message_and_nodes()
  
}

function appendPublic(){
	var l = document.createElement("h5");
    l.innerHTML = "public";
	l.onclick = privateDiscussion;
	l.id = "public";
    document.querySelector("#discussions-box").appendChild(l);
}

$(document).ready(function(){
	appendPublic()	
  $("button#appendPeer").click(function(e){
    var new_peer = prompt("Please enter new peer:", "IP_Address:Port");
	if (!(new_peer == null || new_peer == "")) {
		send("node",new_peer);
	}
  });

  $("#message").keydown(function(e) {
    if (e.keyCode == 13) {
		var txt = document.getElementById("message").value;
		send("message",txt);
    }
  });

  $("button#send").click(function(e){
	var txt = document.getElementById("message").value;
	send("message",txt);
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
	var ID = event.target.id;
	console.log(ID)
	document.getElementById("msg-box").innerHTML = '';
	if(ID ==="public"){
		private_msg = false;
		private_id = ""
	} else{
		private_msg = true;
		private_id = ID;
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
					b.appendChild(document.createTextNode(private_id + " : "))

					var text = document.createTextNode(msg);
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
        var text = document.createTextNode("");
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
      document.querySelector("#discussions-box").innerHTML = "";
	appendPublic()	
      var dataJSON = JSON.parse(data);
      for(x in dataJSON){
		var l = document.createElement("h5");
        l.innerHTML = dataJSON[x.toString()];
		l.onclick = privateDiscussion;
		l.id = dataJSON[x.toString()];
        document.querySelector("#discussions-box").appendChild(l);
      }
    }
  });
}


