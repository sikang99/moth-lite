package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/fasthttp/websocket"
)

var addr = flag.String("addr", "localhost:8080", "http service address")

var upgrader = websocket.Upgrader{} // use default options

func echo(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		log.Printf("recv: %s", message)

		if string(message) == "image" {
			// Create and send image data
			imageData := make([]byte, 640000)
			start := time.Now()
			for b := 255; b >= 0; b-- {
				for row := 0; row < 400; row++ {
					for col := 0; col < 400; col++ {
						imageData[4*col+1600*row+0] = byte(255 - b)
						imageData[4*col+1600*row+1] = byte(255 - b)
						imageData[4*col+1600*row+2] = byte(b)
						a := int(1.85 * (255.0 - float32(b)))
						if row+col < a-20 || row+col > a+20 {
							imageData[4*col+1600*row+3] = 0
						} else {
							imageData[4*col+1600*row+3] = 255
						}
					}
				}
				c.WriteMessage(websocket.BinaryMessage, imageData)
			}
			elapsed := time.Since(start)
			secs := elapsed.Seconds()
			fmt.Printf("Frame rate = %f\n", 256.0/secs)
		} else {
			err = c.WriteMessage(mt, message)
			if err != nil {
				log.Println("write:", err)
				break
			}
		}
	}
}

func home(w http.ResponseWriter, r *http.Request) {
	homeTemplate.Execute(w, "ws://"+r.Host+"/echo")
}

func main() {
	flag.Parse()
	log.SetFlags(0)
	http.HandleFunc("/echo", echo)
	http.HandleFunc("/", home)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

var homeTemplate = template.Must(template.New("").Parse(`
<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<script>
window.addEventListener("load", function(evt) {
    var output = document.getElementById("output");
    var input = document.getElementById("input");
    var ws;
    var print = function(message) {
        var d = document.createElement("div");
        d.innerHTML = message;
        output.appendChild(d);
    };
    document.getElementById("open").onclick = function(evt) {
        if (ws) {
            return false;
        }
        ws = new WebSocket("{{.}}");
		ws.binaryType = 'arraybuffer';
        ws.onopen = function(evt) {
            print("OPEN");
        }
        ws.onclose = function(evt) {
            print("CLOSE");
            ws = null;
        }
        ws.onmessage = function(evt) {
			if (typeof evt.data == "object") {
				draw(evt.data)
			} else {
				print("RESPONSE: " + evt.data);
			}
        }
        ws.onerror = function(evt) {
            print("ERROR: " + evt.data);
        }
        return false;
    };
    document.getElementById("send").onclick = function(evt) {
        if (!ws) {
            return false;
        }
        print("SEND: " + input.value);
        ws.send(input.value);
        return false;
    };
    document.getElementById("close").onclick = function(evt) {
        if (!ws) {
            return false;
        }
        ws.close();
        return false;
    };
});
</script>
</head>
<body>
<table>
<tr><td valign="top" width="50%">
<p>Click "Open" to create a connection to the server, 
"Send" to send a message to the server and "Close" to close the connection. 
You can change the message and send multiple times.
<p>
<form>
<button id="open">Open</button>
<button id="close">Close</button>
<p><input id="input" type="text" value="Hello world!">
<button id="send">Send</button>
</form>
<canvas id="canvas" style="width: 400px; height: 400px; border: 2px solid black;"></canvas>
</td><td valign="top" width="50%">
<div id="output"></div>
</td></tr></table>
</body>
<script>
var canvas = document.getElementById('canvas');
var ctx = canvas.getContext('2d');

function draw(data) {
	var imageData = ctx.createImageData(400, 400);
	var pixels = imageData.data;
	var buffer = new Uint8Array(data);
	for (var i=0; i < pixels.length; i++) {
    	pixels[i] = buffer[i];
	}
  	ctx.putImageData(imageData, 0, 0);
}
</script>
</html>
`))
