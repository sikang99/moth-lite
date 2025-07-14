const recorder = (() => {
    let recorderObject = null;
    const recordedChunks = [];

    return {
        set: (object) => {
            recorderObject = object;
        },
        get: () => {
            return recorderObject;
        },
        push: (chunk) => {
            recordedChunks.push(chunk);
        },
        finished: () => {
            return recordedChunks;
        },
    };
})();

const muxStream = (videoStream, audioStream) => {
    const muxer = new MediaStream();
    muxer.addTrack(videoStream.getVideoTracks()[0]);
    muxer.addTrack(audioStream.getAudioTracks()[0]);
    return muxer;
};

const muxMIME = (videoMIME, audioMIME) => {
    let mime = "video/webm;";
    let videoCodecID = videoMIME.mimeType.match(/^video\/(.+)/)[1];
    if (videoCodecID == "av1") {
        videoCodecID = "av01";
    }
    mime += videoCodecID + ",";
    mime += audioMIME.mimeType.match(/^audio\/(.+)/)[1];
    mime += "\"";
    return mime;
};

export const startRecording = async (
    videoStream,
    audioStream,
    videoMIME,
    audioMIME,
    setSrcObject
) => {
    const muxedStream = muxStream(videoStream, audioStream);
    const muxedMIME = muxMIME(videoMIME, audioMIME);

    console.log(muxedMIME);
    if (MediaRecorder.isTypeSupported(muxedMIME)) {
        console.log("Recording supported");
    } else {
        throw "Unsupported MIME type";
    }

    recorder.set(new MediaRecorder(muxedStream, { mimeType: muxedMIME }));
    recorder.get().addEventListener("dataavailable", (evt) => {
        if (evt.data.size > 0) {
            recorder.push(evt.data);
        }
    });
    recorder.get().start(1000);
    setSrcObject(muxedStream);

    console.log(recorder.get());
};

export const finishRecording = (saveFileCallback) => {
    recorder.get().stop();
    saveFileCallback(recorder.finished());
};
