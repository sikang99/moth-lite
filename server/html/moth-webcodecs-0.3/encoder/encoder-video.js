import { videoMimeAPI } from "./moth-video.js";

const encoder = (() => {
    let encoderObject;

    return {
        set: (videoEncoder) => {
            encoderObject = videoEncoder;
        },
        get: () => {
            return encoderObject;
        },
    };
})();

export const configVideoEncoder = async (videoCodecConfig, stream, sendByte) => {
    const init = {
        output: (chunk) => {
            encoder.pendingOutputs--;
            handleChunk(chunk);
        },
        error: (err) => {
            encoder.keepGoing = false;
            throw err;
        },
    };

    const handleChunk = (chunk) => {
        let chunkData = new Uint8Array(chunk.byteLength);
        chunk.copyTo(chunkData);
        sendByte(chunkData);
    };

    let track = stream.getVideoTracks()[0];
    const trackProcessor = new MediaStreamTrackProcessor(track);

    encoder.reader = trackProcessor.readable.getReader();

    const { supported } = await VideoEncoder.isConfigSupported(videoCodecConfig);
    if (supported) {
        let videoEncoder = new VideoEncoder(init);
        videoEncoder.configure(videoCodecConfig);

        encoder.set(videoEncoder);
        encoder.pendingOutputs = 0;
        encoder.keepGoing = true;
        encoder.frameCounter = 0;
        encoder.keyFrameInterval = videoCodecConfig.interval;

        let mime = new videoMimeAPI();
        mime.parseByCodecConfig(videoCodecConfig);

        console.log(mime.getMimeString(videoCodecConfig));

        sendByte(mime.getMimeString(videoCodecConfig));
    } else {
        throw "Codec Configuration Not supported";
    }
};

export const startVideoEncoding = async () => {
    async function readFrame() {
        const result = await encoder.reader.read();
        let frame = result.value;

        if (frame === undefined) {
            return;
        }

        if (
            encoder.get().state == "configured" &&
            encoder.keepGoing &&
            encoder.pendingOutputs <= 2
        ) {
            encoder.frameCounter++;
            encoder.pendingOutputs++;
            encoder.get().encode(frame, {
                keyFrame: encoder.frameCounter % encoder.keyFrameInterval == 0,
            });
            frame.close();
        } else {
            frame.close();
        }

        setTimeout(readFrame, 1);
    }

    readFrame();
};

export const stopVideoEncoding = async () => {
    try {
        let videoEncoder = encoder.get();
        let frame = await encoder.reader.read().value;
        if (frame != undefined) {
            await encoder.reader.read().value.close();
        }
        await videoEncoder.flush();
        videoEncoder.close();
    } catch (err) {
        throw err;
    }
};
