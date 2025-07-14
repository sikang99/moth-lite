import { moth, mothConfig } from "../moth/moth.js";
import {
    videoMimeAPI,
    webcamObject,
    videoCodecConfig,
} from "../moth/moth-video.js";
import {
    configVideoEncoder,
    startVideoEncoding,
    stopVideoEncoding,
} from "./encoder-video.js";
import {
    audioMimeAPI,
    micObject,
    audioCodecConfig,
} from "../moth/moth-audio.js";
import {
    configAudioEncoder,
    startAudioEncoding,
    stopAudioEncoding,
} from "./encoder-audio.js";
import { configImageEncoder, startImageEncoding } from "./encoder-image.js";

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

    return wsurl;
};

const getBrowserSupportCodecs = () => {
    let videoCodecs = new Set([]);
    let audioCodecs = new Set([]);

    RTCRtpSender.getCapabilities("video").codecs.forEach((videoCodec) => {
        videoCodecs.add(videoCodec.mimeType.match(/^video\/(.+)/)[1]);
    });

    RTCRtpSender.getCapabilities("audio").codecs.forEach((audioCodec) => {
        audioCodecs.add(audioCodec.mimeType.match(/^audio\/(.+)/)[1]);
    });

    return {
        supportedVideoCodecs: Array.from(videoCodecs),
        supportedAudioCodecs: Array.from(audioCodecs),
    };
};

const isConfigured = () => {
    try {
        if (!mothConfig.isConfigured()) {
            throw "Moth Not Configured";
        }

        if (!videoCodecConfig.isConfigured()) {
            throw "Video Codec Not Configured";
        }

        if (!audioCodecConfig.isConfigured()) {
            throw "Audio Codec Not Configured";
        }
    } catch (err) {
        console.error(err);
    }
};

window.onload = () => {
    "use strict";

    let { supportedVideoCodecs, supportedAudioCodecs } =
        getBrowserSupportCodecs();

    $("supported-video-codecs").innerHTML = supportedVideoCodecs.join(", ");
    $("supported-audio-codecs").innerHTML = supportedAudioCodecs.join(", ");

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
                $("name").value == "" ? "webcodecs-test" : $("name").value;
        }

        mothConfig.save(config);
        console.log(mothConfig.get());
    });

    $("save-video-codec-config").addEventListener("click", () => {
        let config = {
            codec:
                $("video-codecs").value == ""
                    ? "vp09.00.31.08"
                    : $("video-codecs").value,
            width: $("video-width").value == "" ? 640 : $("video-width").value,
            height:
                $("video-height").value == "" ? 480 : $("video-height").value,
            interval:
                $("video-interval").value == ""
                    ? 60
                    : $("video-interval").value,
            bitrate:
                $("video-bitrate").value == ""
                    ? 4 * mbps
                    : $("video-bitrate").value * mbps,
            framerate:
                $("video-framerate").value == ""
                    ? 30
                    : $("video-framerate").value,
            latencyMode: "realtime",
        };

        if (config.codec.includes("avc1")) {
            config.avc = { format: "annexb" };
            config.hardwareAcceleration = $("video-hw-acceleration").checked
                ? "prefer-hardware"
                : "prefer-software";
        }

        videoCodecConfig.save(config);
        console.log(videoCodecConfig.get());
    });

    $("save-audio-codec-config").addEventListener("click", () => {
        let config = {
            codec:
                $("audio-codecs").value == ""
                    ? "opus"
                    : $("audio-codecs").value,
            sampleRate:
                $("audio-sampleRate").value == ""
                    ? 48 * kbps
                    : $("audio-sampleRate").value,
            depth: $("audio-depth").value == "" ? 16 : $("audio-depth").value,
            numberOfChannels:
                $("audio-numberOfChannels").value == ""
                    ? 1
                    : $("audio-numberOfChannels").value,
            bitrate:
                $("audio-bitrate").value == ""
                    ? 512 * kbps
                    : $("audio-bitrate").value * kbps,
            latencyMode: "realtime",
        };

        audioCodecConfig.save(config);
        console.log(audioCodecConfig.get());
    });

    $("open-webcam").addEventListener("click", () => {
        const webcam = webcamObject;
        webcam.open(() => {
            $("pub-video").srcObject = webcam.getStream();
            $("pub-image").width = $("pub-video").width;
            $("pub-image").height = $("pub-video").height;
            setTimeout(function canvasUpdate() {
                $("pub-image").getContext("2d").drawImage($("pub-video"), 0, 0);
                setTimeout(
                    canvasUpdate,
                    1000 / (videoCodecConfig.get().framerate * 2)
                );
            }, 1000 / (videoCodecConfig.get().framerate * 2));
            $("close-webcam").addEventListener("click", () => {
                $("pub-video").srcObject = null;
                webcam.close();
            });
        });
    });

    $("open-mic").addEventListener("click", () => {
        const mic = micObject;
        mic.open(() => {
            $("close-mic").addEventListener("click", () => {
                $("pub-audio").srcObject = null;
                mic.close();
            });
        });
    });

    $("start-encoding-publishing").addEventListener("click", async () => {
        isConfigured();

        const pubMime = new videoMimeAPI();
        pubMime.parseByCodecConfig(videoCodecConfig.get());
        const pubAudioMime = new audioMimeAPI();
        pubAudioMime.parseByCodecConfig(audioCodecConfig.get());

        let wsurl = parseMothuri(
            mothConfig.get(),
            pubMime.getMime().mimeType,
            "video",
            "pub"
        );

        let audioWsurl = parseMothuri(
            mothConfig.get(),
            pubAudioMime.getMime().mimeType,
            "audio",
            "pub"
        );

        const webcamStream = webcamObject.getStream();
        const videoConfig = videoCodecConfig.get();

        const micStream = micObject.getStream();
        const audioConfig = audioCodecConfig.get();

        if (webcamObject.isOpen()) {
            await moth.add(wsurl, "webcam-pub", false);
            let mothVideoObject = moth.get("webcam-pub");

            mothVideoObject.onopen = () => {
                if (videoCodecConfig.get().codec.includes("image")) {
                    configImageEncoder(videoConfig, (byte) => {
                        mothVideoObject.send(byte);
                    });
                    startImageEncoding($("pub-image"), (byte) => {
                        mothVideoObject.send(byte);
                    });

                    $("stop-encoding-publishing").addEventListener(
                        "click",
                        async () => {
                            await stopImageEncoding();
                            moth.destroy("webcam-pub");
                        }
                    );
                } else {
                    configVideoEncoder(videoConfig, webcamStream, (byte) => {
                        mothVideoObject.send(byte);
                    })
                        .then(() => {
                            startVideoEncoding()
                                .then()
                                .catch((err) => {
                                    console.error(err);
                                });
                        })
                        .catch((err) => {
                            console.error(err);
                        });
                    $("stop-encoding-publishing").addEventListener(
                        "click",
                        async () => {
                            await stopVideoEncoding();
                            moth.destroy("webcam-pub");
                        }
                    );
                }
            };
        } else {
            throw "Webcam Stream Not Found";
        }

        if (micObject.isOpen()) {
            await moth.add(audioWsurl, "mic-pub");
            let mothAudioObject = moth.get("mic-pub");

            mothAudioObject.onopen = () => {
                configAudioEncoder(audioConfig, micStream, (byte) => {
                    mothAudioObject.send(byte);
                })
                    .then(() => {
                        startAudioEncoding()
                            .then()
                            .catch((err) => {
                                console.error(err);
                            });
                    })
                    .catch((err) => {
                        console.error(err);
                    });
                $("stop-encoding-publishing").addEventListener(
                    "click",
                    async () => {
                        await stopAudioEncoding();
                        moth.destroy("mic-pub");
                    }
                );
            };
        } else {
            throw "Mic Stream Not Found";
        }
    });
    $("stress-test").addEventListener("click", async () => {
        setInterval(() => {
            let array = new Uint32Array(10);
            window.crypto.getRandomValues(array);
            let tmp = 1;
            for (var i = 0; i < array.length; i++) {
                tmp = tmp * array[i];
                console.log(tmp);
            }
        }, 1);
    });
};
