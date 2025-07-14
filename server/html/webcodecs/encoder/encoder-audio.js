import { audioMimeAPI } from "../moth/moth-audio.js";

const encoder = (() => {
    let encoderObject;

    return {
        set: (audioEncoder) => {
            encoderObject = audioEncoder;
        },
        get: () => {
            return encoderObject;
        },
    };
})();

export const configAudioEncoder = async (
    audioCodecConfig,
    stream,
    sendByte
) => {
    const init = {
        output: (chunk) => {
            encoder.pendingOutputs -= 5;
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

    let track = stream.getAudioTracks()[0];
    const trackProcessor = new MediaStreamTrackProcessor(track);

    encoder.reader = trackProcessor.readable.getReader();

    const { supported } = await AudioEncoder.isConfigSupported(
        audioCodecConfig
    );
    if (supported) {
        let audioEncoder = new AudioEncoder(init);
        audioEncoder.configure(audioCodecConfig);

        encoder.set(audioEncoder);
        encoder.pendingOutputs = 0;
        encoder.keepGoing = true;

        let mime = new audioMimeAPI();
        mime.parseByCodecConfig(audioCodecConfig);

        console.log(mime.getMimeString(audioCodecConfig));

        sendByte(mime.getMimeString(audioCodecConfig));
    } else {
        throw "Codec Configuration Not supported";
    }
};

export const startAudioEncoding = async () => {
    async function readFrame() {
        const result = await encoder.reader.read();
        let frame = result.value;

        if (frame === undefined) {
            return;
        }

        if (
            encoder.get().state == "configured" &&
            encoder.keepGoing &&
            encoder.pendingOutputs < 1
        ) {
            encoder.pendingOutputs++;
            encoder.get().encode(frame);
            frame.close();
        } else {
            encoder.pendingOutputs--;
            frame.close();
        }

        setTimeout(readFrame, 1);
    }

    readFrame();
};

export const stopAudioEncoding = async () => {
    try {
        let audioEncoder = encoder.get();
        let frame = await encoder.reader.read().value;
        if (frame != undefined) {
            await encoder.reader.read().value.close();
        }
        await audioEncoder.flush();
        audioEncoder.close();
    } catch (err) {
        throw err;
    }
};
