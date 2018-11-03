
function send(type) {
  var txt;
 
  if (type === "message"){
    txt = document.getElementById("message").value;
  } else{
    txt = document.getElementById("messagePeer").value;
  }
  if (txt.length > 0) {
    $.ajax({
      type: "POST",
      url: "http://localhost:8080/"+type,
      data: {value:txt}
    })
    .done(function(data) {
      if (type === "message"){
        document.getElementById("message").value = '';
      } else {
        document.getElementById("messagePeer").value = '';
      }
    });
  }
  
  get_message_and_nodes()
  
}


$(document).ready(function(){
  $("button#send").click(function(e){
    send("message");
  });

  $("button#sendPeer").click(function(e){
    send("node");
  });

  $("#message").keydown(function(e) {
    if (e.keyCode == 13) {
      send("message");
    }
  });

  $("#messagePeer").keydown(function(e) {
    if (e.keyCode == 13) {
      send("node");
    }
  });
);
  add_peer_btn = document.getElementById("appendPeer");
add_peer_btn
});

$.ajax({
  type: "GET",
  url: "http://localhost:8080/id",
  success: function(data,status,xhr){
    var p = document.createElement("p");
    var dataJSON = JSON.parse(data);
    var text = document.createTextNode("ID : " + dataJSON.toString());
    p.appendChild(text);
    document.querySelector("#id-box").appendChild(text);
  }
});

get_message_and_nodes();

setInterval(get_message_and_nodes, 1000);
function get_message_and_nodes(){
  $.ajax({
    type: "GET",
    url: "http://localhost:8080/message",
    success: function(data,status,xhr){
      var out = document.querySelector("#msg-box");

      const isScrolledToBottom = out.scrollHeight - out.clientHeight <= out.scrollTop + 1

      document.querySelector("#msg-box").innerHTML = "";
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
        out.scrollTop = out.scrollHeight - out.clientHeight
      }

    }
  });

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
}


