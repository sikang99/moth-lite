export const moth = (() => {
    let container = [];

    const mothObject = {
        wsObject: null,

        async connect(wsurl) {
            try {
                this.wsObject = new WebSocket(wsurl);
            } catch {
                throw "Error on connecting websocket";
            }

            // this.wsObject.onopen = function () {
            //     console.log(wsurl + " opened");
            // };

            this.wsObject.onclose = function (event) {
                console.log(event);
            };

            return this.wsObject;
        },

        async close() {
            if (this.wsObject.bufferedAmount != 0) {
                // flush buffer
                this.wsObject.close();
            } else {
                this.wsObject.close();
            }
        },
    };

    const createMothObject = async (url, name) => {
        if (
            container.find((mothObject) => mothObject.url == url) !== undefined
        ) {
            throw "Same URL Moth Object Already Allocated";
        } else if (
            container.find((mothObject) => mothObject.name == name) !==
            undefined
        ) {
            throw "Same Name Moth Object Already Allocated";
        }

        console.log(url);
        let obj = {};
        obj.wsObject = await mothObject.connect(url);
        obj.name = name;
        obj.url = url;
        container.push(obj);
    };

    const listMothContainer = () => {
        return container;
    };

    const closeMothObject = (name) => {
        if (name === undefined) {
            throw "Name is undefined";
        } else {
            try {
                const idx = container.findIndex(
                    (mothObject) => mothObject.name == name
                );

                container[idx].wsObject.close();
                container.splice(idx, 1);
            } catch {
                throw "Moth Object Not Found";
            }
        }
    };

    const getMothObject = (name) => {
        if (name === undefined) {
            throw "Name is undefined";
        } else {
            try {
                return container.find((mothObject) => mothObject.name == name)
                    .wsObject;
            } catch {
                throw "Moth Object Not Found";
            }
        }
    };

    return {
        add: createMothObject,
        list: listMothContainer,
        get: getMothObject,
        destroy: closeMothObject,
    };
})();

export const mothConfig = (() => {
    let mothConfig = {};
    let isConfigured = false;

    function publicGetConfig() {
        return mothConfig;
    }

    function privateSetConfig(config) {
        mothConfig = config;
        isConfigured = true;
    }

    function publicSaveConfig(config) {
        privateSetConfig(config);
    }

    return {
        get: publicGetConfig,
        save: publicSaveConfig,
        isConfigured: () => {
            return isConfigured;
        },
    };
})();
