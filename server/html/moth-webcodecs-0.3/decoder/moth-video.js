export const videoMimeAPI = function () {
    let mimeObject = {};
    let isConfigured = false;

    const privateSetMime = (mimeType, mimeOption) => {
        mimeObject = {
            mimeType: mimeType,
            mimeOption: mimeOption,
        };

        if (!isConfigured) {
            isConfigured = true;
        }
    };

    const publicParseByCodecConfig = (codecConfig) => {
        let mimeType;

        if (codecConfig.codec.includes("avc1")) {
            mimeType = "video/h264";
        } else if (codecConfig.codec.includes("vp8")) {
            mimeType = "video/vp8";
        } else if (codecConfig.codec.includes("vp09")) {
            mimeType = "video/vp9";
        } else if (codecConfig.codec.includes("av01")) {
            mimeType = "video/av1";
        }

        let mimeOption = {};

        if (codecConfig.width) {
            mimeOption.width = codecConfig.width;
        }
        if (codecConfig.height) {
            mimeOption.height = codecConfig.height;
        }
        if (codecConfig.interval) {
            mimeOption.interval = codecConfig.interval;
        }
        if (codecConfig.bitrate) {
            mimeOption.bitrate = codecConfig.bitrate;
        }
        if (codecConfig.framerate) {
            mimeOption.framerate = codecConfig.framerate;
        }

        privateSetMime(mimeType, mimeOption);
    };

    const publicCreateCodecConfigFromMimeObject = () => {
        let codecConfig = {};

        if ((mimeObject.mimeType == "video/vp9")) {
            codecConfig.codec = "vp09.00.31.08";
        } else if ((mimeObject.mimeType == "video/h264")) {
            codecConfig.codec = "avc1.64001E";
        } else if ((mimeObject.mimeType == "video/vp8")) {
            codecConfig.codec = "vp8";
        } else if ((mimeObject.mimeType == "video/av1")) {
            codecConfig.codec = "av01.0.05M.10";
        }

        if (mimeObject.mimeOption.width) {
            codecConfig.width = mimeObject.mimeOption.width;
        }
        if (mimeObject.mimeOption.height) {
            codecConfig.height = mimeObject.mimeOption.height;
        }
        if (mimeObject.mimeOption.interval) {
            codecConfig.interval = mimeObject.mimeOption.interval;
        }
        if (mimeObject.mimeOption.bitrate) {
            codecConfig.bitrate = mimeObject.mimeOption.bitrate;
        }
        if (mimeObject.mimeOption.framerate) {
            codecConfig.framerate = mimeObject.mimeOption.framerate;
        }

        return codecConfig;
    };

    const publicParseFromMimeString = (mimeString) => {
        let mimeSplited = mimeString.split(";");

        let mimeType = mimeSplited.shift().trim();

        let mimeOption = {};
        mimeSplited.forEach((mimeParams) => {
            let mimeKeyValue = mimeParams.trim().split("=");
            mimeOption[mimeKeyValue[0]] = mimeKeyValue[1];
        });

        privateSetMime(mimeType, mimeOption);
    };

    const publicGetMime = () => {
        return mimeObject;
    };

    const publicCreateMimeStringFromMimeObject = () => {
        let mimeString = mimeObject.mimeType + ";";

        if (mimeObject.mimeOption.width) {
            mimeString += "width=" + mimeObject.mimeOption.width + ";";
        }
        if (mimeObject.mimeOption.height) {
            mimeString += "height=" + mimeObject.mimeOption.height + ";";
        }
        if (mimeObject.mimeOption.framerate) {
            mimeString += "framerate=" + mimeObject.mimeOption.framerate + ";";
        }
        if (mimeObject.mimeOption.interval) {
            mimeString += "interval=" + mimeObject.mimeOption.interval + ";";
        }
        if (mimeObject.mimeOption.bitrate) {
            mimeString += "bitrate=" + mimeObject.mimeOption.bitrate + ";";
        }

        mimeString += "packet=stream";

        return mimeString;
    };

    return {
        parseByCodecConfig: publicParseByCodecConfig,
        parseMimeString: publicParseFromMimeString,
        getMime: publicGetMime,
        getMimeString: publicCreateMimeStringFromMimeObject,
        getCodecConfig: publicCreateCodecConfigFromMimeObject,
        isConfigured: () => {
            return isConfigured;
        },
    };
};

export const webcamObject = (() => {
    let stream = null;
    let isOpen = false;

    const publicOpenWebCamStream = (fallback, err) => {
        if (navigator.mediaDevices.getUserMedia) {
            navigator.mediaDevices
                .getUserMedia({ video: true })
                .then(function (streamSrc) {
                    stream = streamSrc;
                    fallback();
                    isOpen = true;
                    return true;
                })
                .catch(function (err) {
                    console.log(err);
                    return false;
                });
        } else {
            if (err) {
                throw err;
            } else {
                throw "Webcam API is not supported in this browser.";
            }
        }
    };

    const publicCloseWebcamStream = () => {
        if (stream) {
            stream.getTracks().forEach((track) => {
                track.stop();
            });
            isOpen = false;
        } else {
            throw "stream is not defined";
        }
    };

    return {
        open: publicOpenWebCamStream,
        close: publicCloseWebcamStream,
        getStream: () => {
            return stream;
        },
        isOpen: () => {
            return isOpen;
        },
    };
})();

export const videoCodecConfig = (() => {
    let codecConfig = {};
    let isConfigured = false;

    const publicGetConfig = () => {
        return codecConfig;
    };

    const privateSetConfig = (config) => {
        codecConfig = config;
        isConfigured = true;
    };

    const publicSaveConfig = (config) => {
        privateSetConfig(config);
    };

    return {
        get: publicGetConfig,
        save: publicSaveConfig,
        isConfigured: () => {
            return isConfigured;
        },
    };
})();
