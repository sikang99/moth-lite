import { videoMimeAPI } from "../moth/moth-video.js";

const encoder = (() => {
    let _mimeString;
    let _codecConfig;
    let timeOutEventID = undefined;

    return {
        setCodec: (codecConfig) => {
            _codecConfig = codecConfig;
        },
        setMime: (mimeString) => {
            _mimeString = mimeString;
        },
        getMime: () => {
            return _mimeString;
        },
        getCodec: () => {
            return _codecConfig;
        },
        getFrame: (canvasDOM) => {
            let promise = new Promise((resolve, reject) => {
                window.setTimeout(() => {
                    resolve(canvasDOM.toDataURL(_codecConfig.codec));
                }, 1000 / _codecConfig.framerate);
            });

            return promise;
        },
    };
})();

export const configImageEncoder = (videoCodecConfig, sendByte) => {
    encoder.setCodec(videoCodecConfig);
    let mime = new videoMimeAPI();
    mime.parseByCodecConfig(videoCodecConfig);
    const mimeString = mime.getMimeString(videoCodecConfig);
    const imageCodecList = ["jpeg", "png", "webp", "avif"];
    if (imageCodecList.some((imageCodec) => mimeString.includes(imageCodec))) {
        console.log(mimeString);
        sendByte(mimeString);
        encoder.setMime(mimeString);
        return;
    }
};

export const startImageEncoding = (canvasDOM, sendByte) => {
    async function readFrame() {
        const frame = await encoder.getFrame(canvasDOM);
        sendByte(frame);
        setTimeout(readFrame, 1);
    }

    readFrame();
};

export const stopImageEncoding = () => {};
