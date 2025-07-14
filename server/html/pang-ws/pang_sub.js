window.addEventListener("load", function(evt) {
    var output = document.getElementById("output");
    var input = document.getElementById("input");
    var ws;


    var print = function(message) { //output 자식노드로 div 생성.
        var d = document.createElement("div");
        d.innerHTML = message;
        output.appendChild(d);
    };
    document.getElementById("open").onclick = function(evt) {
        if (ws) {
            return false;
        }
        //url 조정
        let wsurl;
        const APIOption = {
            mode : "sub",
            channel : "btlb3pjpc98lsdbc0lj0",
            track : "video"
        }
        //const cojamUrl = 'cobot.center'
        let host = window.location.hostname 
        //host = cojamUrl;

        if (window.location.protocol == 'https:') {
            wsurl = `wss://` + host + `:8277/pang/ws/${APIOption.mode}?channel=${APIOption.channel}&track=${APIOption.track}`;
        } else {
            wsurl = `ws://` + host + `:8276/pang/ws/${APIOption.mode}?channel=${APIOption.channel}&track=${APIOption.track}`;
        }
        //url 조정

        ws = new WebSocket(wsurl);
        ws.binaryType = 'arraybuffer';

        ws.onopen = function(evt) {
            print("OPEN");
        }
        ws.onclose = function(evt) {
            print("CLOSE");
            ws = null;
        }
        ws.onmessage = function(evt) {
            console.log(evt)
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
    
    // document.getElementById("send").onclick = function(evt) {
    //     if (!ws) {
    //         return false;
    //     }
    //     print("SEND: " + input.value);
    //     ws.send(input.value);
    //     return false;
    // };
    document.getElementById("close").onclick = function(evt) {
        if (!ws) {
            return false;
        }
        ws.close();
        return false;
    };
});
