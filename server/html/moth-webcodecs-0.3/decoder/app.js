import { moth, mothConfig } from "./moth.js";
import {
    configVideoDecoder,
    decodeVideoFrame,
    stopVideoDecoding,
} from "./decoder-video.js";
import {
    configAudioDecoder,
    decodeAudioFrame,
    stopAudioDecoding,
} from "./decoder-audio.js";
import { startRecording, finishRecording } from "./recorder.js";
import { audioMimeAPI } from "./moth-audio.js";
import { videoMimeAPI } from "./moth-video.js";

const kbps = 1000;
const mbps = kbps * kbps;

const $ = function (id) {
    return document.getElementById(id);
};

export const parseMothuri = (mothConfig, mimeType, track, type) => {
    let wsurl = mothConfig.dest + mothConfig.api;
    wsurl += "/" + type + "?";
    wsurl += "channel=" + mothConfig.channel;
    if (mothConfig.name) {
        wsurl += "&name=" + mothConfig.name;
    }
    wsurl += "&track=" + track;
    wsurl += "&mime=";
    wsurl += type == "pub" ? mimeType : "sendtome";

    return wsurl;
};

window.onload = () => {
    "use strict";

    let videoTrack = null;
    let audioTrack = null;
    let videoMIME = null;
    let audioMIME = null;

    $("save-moth-config").addEventListener("click", () => {
        let config = {
            dest:
                $("dest").value == ""
                    ? "wss://cobot.center:8287"
                    : $("dest").value,
            api: $("api").value == "" ? "/pang/ws" : $("api").value,
            channel: $("channel").value == "" ? "instant" : $("channel").value,
        };

        if (config.channel == "instant") {
            config.name =
                $("name").value == "" ? "webcodecs-test" : $("name").valueName;
        }

        mothConfig.save(config);
        console.log(mothConfig.get());
    });

    $("start-decoding-subscribing").addEventListener("click", async () => {
        let wsurl = parseMothuri(mothConfig.get(), "sendtome", "video", "sub");
        let audioWsurl = parseMothuri(
            mothConfig.get(),
            "sendtome",
            "audio",
            "sub"
        );

        await moth.add(wsurl, "webcam-sub");
        await moth.add(audioWsurl, "mic-sub");
        let mothObject = moth.get("webcam-sub");
        let audioMothObject = moth.get("mic-sub");

        mothObject.onopen = () => {
            const drawFrame = (frame) => {
                $("sub-video").getContext("2d").drawImage(frame, 0, 0);
            };

            const drawFPS = (fps) => {
                $("fps").innerHTML = `${fps}`;
            };
            configVideoDecoder(drawFrame, drawFPS, (generator) => {
                videoTrack = new MediaStream([generator]);
            })
                .then(() => {
                    mothObject.onmessage = (bytestream) => {
                        decodeVideoFrame(bytestream, (codecConfig) => {
                            let vMimeAPI = new videoMimeAPI();
                            vMimeAPI.parseByCodecConfig(codecConfig);
                            videoMIME = vMimeAPI.getMime();
                            $("detected-video-codec").innerHTML =
                                JSON.stringify(codecConfig);
                        })
                            .then(() => {})
                            .catch((err) => {
                                console.error(err);
                            });
                    };
                    $("stop-decoding-subscribing").addEventListener(
                        "click",
                        async () => {
                            await stopVideoDecoding();
                            moth.destroy("webcam-sub");
                        }
                    );
                })
                .catch((err) => {
                    console.error(err);
                });
        };

        audioMothObject.onopen = () => {
            configAudioDecoder((generator) => {
                audioTrack = new MediaStream([generator]);
                $("sub-audio").srcObject = audioTrack;
            }).then(() => {
                audioMothObject.onmessage = (bytestream) => {
                    decodeAudioFrame(bytestream, (codecConfig) => {
                        let aMimeAPI = new audioMimeAPI();
                        aMimeAPI.parseByCodecConfig(codecConfig);
                        audioMIME = aMimeAPI.getMime();
                        $("detected-audio-codec").innerHTML =
                            JSON.stringify(codecConfig);
                    })
                        .then(() => {})
                        .catch((err) => {
                            console.error(err);
                        });
                };
                $("stop-decoding-subscribing").addEventListener(
                    "click",
                    async () => {
                        await stopAudioDecoding();
                        moth.destroy("mic-sub");
                    }
                );
                $("mute-audio").addEventListener("click", () => {
                    if ($("sub-audio").muted == true) {
                        $("sub-audio").muted = false;
                        $("mute-audio").innerHTML = "MUTE";
                    } else {
                        $("sub-audio").muted = true;
                        $("mute-audio").innerHTML = "UNMUTE";
                    }
                });
            });
        };
    });

    $("start-recording").addEventListener("click", async () => {
        startRecording(videoTrack, audioTrack, videoMIME, audioMIME, (muxedStream) => {
                $("test").srcObject = muxedStream;
            }).then(() => {
                $("save-recording").addEventListener("click", () => {
                    finishRecording((recordedChunks) => {
                        const blob = new Blob(recordedChunks);
                        console.log(blob);
                        $("download-record").href =
                            window.URL.createObjectURL(blob);;
                        $("download-record").download = "record.webm";
                        $("download-record").innerHTML =
                            "download record.webm";
                        $("download-record").click();
                    });
                });
            }
        );
    });
};
