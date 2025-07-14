import { videoMimeAPI } from "../moth/moth-video.js";

const decoder = (() => {
    let decoderObject = null;
    let mime = null;
    let isKeyFrameRecv = false;
    let init = {};

    const createVideoDecoder = async (videoCodecConfig) => {
        const { supported } = await VideoEncoder.isConfigSupported(
            videoCodecConfig
        );
        if (supported) {
            let videoDecoder = new VideoDecoder(init);
            videoDecoder.configure(videoCodecConfig);

            decoderObject = videoDecoder;
            isKeyFrameRecv = false;
        } else {
            console.log("Codec Configuration Not supported");
            return;
        }
    };

    return {
        set: createVideoDecoder,
        get: () => {
            return decoderObject;
        },
        setMime: (mimeObject) => {
            mime = mimeObject;
        },
        getMime: () => {
            return mime;
        },
        setInit: (initObject) => {
            init = initObject;
        },
    };
})();

const videoAPI = (() => {
    let drawFrame;
    let drawFPS;
    let startTime = 0.0;
    let pendingFrames = [];
    let paintCount = 0;
    let underflow = true;
    let timeBase = 0;

    const generator = new MediaStreamTrackGenerator({
        kind: "video",
    });
    const { writable } = generator;
    const videoWriter = writable.getWriter();

    const publicRenderFrame = async () => {
        if (startTime === 0.0) {
            startTime = performance.now();
        }

        underflow = pendingFrames.length == 0;
        if (underflow) return;

        const frame = pendingFrames.shift();
        await new Promise((r) => {
            setTimeout(r, privateCalculateTimeUntilNextFrame(frame.timestamp));
        });

        const recordFrame = frame.clone();
        drawFrame(frame);
        await videoWriter.write(recordFrame);

        frame.close();

        const elapsed = (performance.now() - startTime) / 1000.0;
        const fps = (++paintCount / elapsed).toFixed(3);
        drawFPS(fps);

        setTimeout(publicRenderFrame, 0);
    };

    const privateCalculateTimeUntilNextFrame = (timestamp) => {
        if (timeBase == 0) timeBase = performance.now();
        let mediaTime = performance.now() - timeBase;
        return Math.max(0, timestamp / 1000 - mediaTime);
    };

    return {
        set: (drawFrameFunc, drawFPSFunc) => {
            drawFrame = drawFrameFunc;
            drawFPS = drawFPSFunc;
        },
        pushPendingFrames: (frame) => {
            pendingFrames.push(frame);
        },
        isUnderFlow: () => {
            return underflow;
        },
        render: publicRenderFrame,
        getGenerator: () => {
            return generator;
        },
    };
})();

export const configVideoDecoder = async (drawFrame, drawFPS, setSrcObject) => {
    const handleFrame = (frame) => {
        videoAPI.pushPendingFrames(frame);
        if (videoAPI.isUnderFlow()) {
            setTimeout(videoAPI.render, 0);
        }
    };

    const init = {
        output: handleFrame,
        error: (err) => {
            throw err;
        },
    };

    decoder.setInit(init);
    videoAPI.set(drawFrame, drawFPS);
    setSrcObject(videoAPI.getGenerator());
};

export const decodeVideoFrame = async (bytestream, onMimeChangeCallback) => {
    if (typeof bytestream.data === "string") {
        if (bytestream.data.includes("image")) {
            var image = new Image();
            image.onload = () => {
                document.getElementById("sub-video").getContext("2d").drawImage(image, 0, 0);
            }
            image.src = bytestream.data;
        } else {
            const mimeAPI = new videoMimeAPI();
            mimeAPI.parseMimeString(bytestream.data);
            decoder.setMime(mimeAPI.getMime());
            const codecConfig = mimeAPI.getCodecConfig();
            console.log("Video Codec : ", codecConfig);
            decoder.set(codecConfig);
            onMimeChangeCallback(codecConfig);
        }
    } else {
        bytestream.data.arrayBuffer().then((videoArrayBuffer) => {
            const chunk = new EncodedVideoChunk({
                timestamp: bytestream.timeStamp,
                type: bytestream.type ? "key" : "delta",
                data: videoArrayBuffer,
                duration: null,
            });

            if (chunk.type === "delta") {
                if (!decoder.isKeyFrameRecv) return;
            } else {
                decoder.isKeyFrameRecv = true;
            }

            decoder.get().decode(chunk);
        });
    }
};

export const stopVideoDecoding = async () => {
    await decoder.get().flush();
    decoder.get().close();
};
