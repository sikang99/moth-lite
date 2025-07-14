export const audioMimeAPI = function () {
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

    const publicParseByCodecConfig = (audioCodecConfig) => {
        let mimeType;

        if (audioCodecConfig.codec.includes("opus")) {
            mimeType = "audio/opus";
        }

        let mimeOption = {};

        if (audioCodecConfig.sampleRate) {
            mimeOption.rate = audioCodecConfig.sampleRate;
        }
        if (audioCodecConfig.depth) {
            mimeOption.depth = audioCodecConfig.depth;
        }
        if (audioCodecConfig.numberOfChannels) {
            mimeOption.channels = audioCodecConfig.numberOfChannels;
        }
        if (audioCodecConfig.bitrate) {
            mimeOption.bitrate = audioCodecConfig.bitrate;
        }

        privateSetMime(mimeType, mimeOption);
    };

    const publicCreateCodecConfigFromMimeObject = () => {
        let audioCodecConfig = {};

        if (mimeObject.mimeType == "audio/opus") {
            audioCodecConfig.codec = "opus";
        }

        if (mimeObject.mimeOption.rate) {
            audioCodecConfig.sampleRate = mimeObject.mimeOption.rate;
        }
        if (mimeObject.mimeOption.depth) {
            audioCodecConfig.depth = mimeObject.mimeOption.depth;
        }
        if (mimeObject.mimeOption.channels) {
            audioCodecConfig.numberOfChannels = mimeObject.mimeOption.channels;
        }
        if (mimeObject.mimeOption.bitrate) {
            audioCodecConfig.bitrate = mimeObject.mimeOption.bitrate;
        }

        return audioCodecConfig;
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

        if (mimeObject.mimeOption.rate) {
            mimeString += "rate=" + mimeObject.mimeOption.rate + ";";
        }
        if (mimeObject.mimeOption.depth) {
            mimeString += "depth=" + mimeObject.mimeOption.depth + ";";
        }
        if (mimeObject.mimeOption.channels) {
            mimeString += "channels=" + mimeObject.mimeOption.channels + ";";
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

export const micObject = (() => {
    let stream = null;
    let isOpen = false;

    const publicOpenMicStream = (fallback, err) => {
        if (navigator.mediaDevices.getUserMedia) {
            navigator.mediaDevices
                .getUserMedia({ audio: true })
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
                throw "Mic API is not supported in this browser.";
            }
        }
    };

    const publicCloseMicStream = () => {
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
        open: publicOpenMicStream,
        close: publicCloseMicStream,
        getStream: () => {
            return stream;
        },
        isOpen: () => {
            return isOpen;
        },
    };
})();

export const audioCodecConfig = (() => {
    let audioCodecConfig = {};
    let isConfigured = false;

    const publicGetConfig = () => {
        return audioCodecConfig;
    };

    const privateSetConfig = (config) => {
        audioCodecConfig = config;
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
