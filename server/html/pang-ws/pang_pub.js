window.addEventListener("load", function(evt) {
    var output = document.getElementById("output");
    var ws;
    const video = document.getElementById('video');
    const imageFormats = document.getElementById("imageFormats");
    const resolution = document.getElementById("resolution");
    const formet = document.getElementById("formet");
    const canvasDisplay = document.getElementById('canvasDisplay');
    const snapshotDisplay = document.getElementById("snapshotDisplay");

    // let snapshot = document.getElementById('snapshot');
    var canvas = document.getElementById('canvas');
    var ctx = canvas.getContext('2d');
    let userStream;
    let value;
    let resolution_value;
    let formet_value;
    let imgFormet = 'jpeg'; //image 포맷
    let img = new Image;
    // 디폴트 해상도
    canvas.width = 640;
    canvas.height = 400;

    // let imgData = {
    //     FrameType : 0,
    //     Mime : "image/jpeg",
    //     // Header : {"Content-Length":[img.length],"Content-":["image/jpeg"]},
    //     Length : 0,
    //     Data : null
    // };
    var print = function(message) { //output 자식노드로 div 생성.
        var d = document.createElement("div");
        d.innerHTML = message;
        output.appendChild(d);
    };
    document.getElementById("open").onclick = function(evt) {//open 클릭
        if (ws) {
            return false;
        }
        //url 조정
        let wsurl;
        const APIOption = {
            mode : "pub",
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
        ws.onerror = function(evt) {
            print("ERROR: " + evt.data);
        }
        return false;
    };

    imageFormats.onchange = (e) =>{ // imageFormat을 선택한 경우 실행
        value = imageFormats.options[imageFormats.selectedIndex].value
        switch(value) { //선택한 값.
            case 'Helloworld':
                if (userStream) {
                    var track = userStream.getTracks()[0];
                    track.stop();
                    video.pause();
                    video.src = "";
                }
                snapshotDisplay.style.display = 'none';
                canvasDisplay.style.display = 'none';
                break;

            case 'Stream':
                canvasDisplay.style.display = 'block';
                snapshotDisplay.style.display = 'none';
                navigator.mediaDevices.getUserMedia({video:true, audio:false})
                .then(function(stream) {
                /* 스트림 사용 */
                    video.srcObject = stream;
                    userStream = stream;
                    video.play();
                })
                .catch(function(err) {
                /* 오류 처리 */
                console.log("에러 : ",err)
                });
                break;

            case 'Snapshot':
            if (userStream) {
                var track = userStream.getTracks()[ 0 ];
                track.stop();
                video.pause();
                video.src = "";
                clearInterval(interval)
            }


            snapshotDisplay.style.display = 'block';
            canvasDisplay.style.display = 'none';
            
            navigator.mediaDevices.getUserMedia({video:true, audio:true})
            .then(function(stream) {
            /* 스트림 사용 */
                video.srcObject = stream;
                userStream = stream;
                video.play();
            })
            .catch(function(err) {
            /* 오류 처리 */
            console.log("에러 : ",err)
            });
            break;
        }
    }
    formet.onchange = (e) => { // 이미지 확장자 변환
        formet_value = formet.options[formet.selectedIndex].value;
        switch(formet_value) {
            case 'default' :
                imgFormet = 'jpeg';
                break;
            case 'png' :
                imgFormet = 'png';
                break;
        }
    }
    resolution.onchange = (e) => { //해상도교체 이벤트
        resolution_value = resolution.options[resolution.selectedIndex].value;
        switch(resolution_value) {
            case 'default' :
                canvas.width = 640;
                canvas.height = 400;
                break;
            case 'HD' :
                canvas.width = 1280;
                canvas.height = 720;
                break;
            case 'FHD' :
                canvas.width = 1920;
                canvas.height = 1080;
                break;
            case 'QHD' :
                canvas.width = 2560;
                canvas.height = 1440;
                break;
            case '4UHD' :
                canvas.width = 3840;
                canvas.height = 2160;
                break;
            case '8UHD' :
                canvas.width = 7680;
                canvas.height = 4320;
                break;
        }
    }
    let interval;
    document.getElementById("send").onclick = function(evt) { //send 클릭 시.
        value = imageFormats.options[imageFormats.selectedIndex].value
        switch(value) { //선택한 값.
            case 'Helloworld':
                ws.send(video.srcObject);
                break;
            case 'Stream':  
            interval = setInterval(() => {
                    draw();
                }, 60);
                break;
            case 'Snapshot':
                draw();
                break;
        }
        
        print("SEND: " + value);
        return false;
    };

    document.getElementById("close").onclick = function(evt) {
        if (!ws) {
            return false;
        }
        ws.close();
        return false;
    };


    const draw = (arg) => {// 영상보내기

        ctx.drawImage(video,0,0,canvas.width, canvas.height);
        
        img.src		= canvas.toDataURL( `image/${imgFormet}` );
        img.width	= canvas.width;
        // snapshot.innerHTML = '';
        // snapshot.appendChild( img );
                
        if ( ws ) ws.send(dataURItoBlob(img.src));
        
    }

    function dataURItoBlob(dataURI) {
        // convert base64 to raw binary data held in a string
        // doesn't handle URLEncoded DataURIs - see SO answer #6850276 for code that does this
        var byteString = atob(dataURI.split(',')[1]);
      
        // separate out the mime component
        var mimeString = dataURI.split(',')[0].split(':')[1].split(';')[0]
      
        // write the bytes of the string to an ArrayBuffer
        var ab = new ArrayBuffer(byteString.length);
      
        // create a view into the buffer
        var ia = new Uint8Array(ab);
      
        // set the bytes of the buffer to the correct values
        for (var i = 0; i < byteString.length; i++) {
            ia[i] = byteString.charCodeAt(i);
        }
      
        // write the ArrayBuffer to a blob, and you're done
        var blob = new Blob([ab], {type: mimeString}); //보낼 객체.
        return blob;
      
      }
});
