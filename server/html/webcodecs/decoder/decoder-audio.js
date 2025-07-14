import { audioMimeAPI } from "../moth/moth-audio.js";

const decoder = (() => {
    let decoderObject = null;
    let mime = null;
    let init = {};
    let source = null;

    const createAudioDecoder = async (audioCodecConfig) => {
        const { supported } = await AudioDecoder.isConfigSupported(
            audioCodecConfig
        );
        if (supported) {
            let audioDecoder = new AudioDecoder(init);
            audioDecoder.configure(audioCodecConfig);

            decoderObject = audioDecoder;
        } else {
            console.log("Codec Configuration Not supported");
            return;
        }
    };

    return {
        set: createAudioDecoder,
        get: () => {
            return decoderObject;
        },
        getSource: () => {
            return source;
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

const audioAPI = (() => {
    let startTime = 0.0;
    let pendingFrames = [];
    let underflow = true;
    let timeBase = 0;

    const generator = new MediaStreamTrackGenerator({
        kind: "audio",
    });
    const { writable } = generator;
    const audioWriter = writable.getWriter();

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
        
        await audioWriter.write(frame);
        frame.close();

        setTimeout(publicRenderFrame, 0);
    };

    const privateCalculateTimeUntilNextFrame = (timestamp) => {
        if (timeBase == 0) timeBase = performance.now();
        let mediaTime = performance.now() - timeBase;
        return Math.max(0, timestamp / 1000 - mediaTime);
    };

    return {
        getGenerator: () => {
            return generator;
        },
        pushPendingFrames: (frame) => {
            pendingFrames.push(frame);
        },
        isUnderFlow: () => {
            return underflow;
        },
        render: publicRenderFrame,
    };
})();

export const configAudioDecoder = async (setSrcObject) => {
    const handleFrame = (frame) => {
        audioAPI.pushPendingFrames(frame);
        if (audioAPI.isUnderFlow()) {
            setTimeout(audioAPI.render, 0);
        }
    };

    const init = {
        output: handleFrame,
        error: (err) => {
            throw err;
        },
    };

    decoder.setInit(init);
    setSrcObject(audioAPI.getGenerator());
};

export const decodeAudioFrame = async (bytestream, onMimeChangeCallback) => {
    if (typeof bytestream.data === "string") {
        const aMimeAPI = new audioMimeAPI();
        aMimeAPI.parseMimeString(bytestream.data);
        const codecConfig = aMimeAPI.getCodecConfig();
        console.log("Audio Codec : ", codecConfig);
        decoder.set(codecConfig);
        onMimeChangeCallback(codecConfig);
    } else {
        bytestream.data.arrayBuffer().then((audioArrayBuffer) => {
            const chunk = new EncodedAudioChunk({
                timestamp: bytestream.timeStamp,
                type: bytestream.type ? "key" : "delta",
                data: audioArrayBuffer,
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

export const stopAudioDecoding = async () => {
    await decoder.get().flush();
    decoder.get().close();
};
