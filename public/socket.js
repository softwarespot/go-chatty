export const LOG_LEVEL_ERROR = 1;
export const LOG_LEVEL_DEBUG = 2;

// Idea based on URL: https://github.com/socketio/socket.io/blob/main/examples/basic-websocket-client/src/index.js
export class Socket {
    #subscribers = new Map();
    #emit(event, ...args) {
        if (!this.#subscribers.has(event)) {
            return;
        }

        const fns = this.#subscribers.get(event);
        for (const fn of fns) {
            fn(...args);
        }
    }
    #on(event, fn) {
        if (!this.#subscribers.has(event)) {
            this.#subscribers.set(event, [fn]);
            return;
        }

        const fns = this.#subscribers.get(event);
        fns.push(fn);
    }
    #off(event, fn) {
        if (event === undefined && fn === undefined) {
            this.#subscribers.clear();
            return;
        }

        if (!this.#subscribers.has(event)) {
            return;
        }

        if (fn === undefined) {
            this.#subscribers.set(event, []);
            return;
        }

        const fns = this.#subscribers.get(event);

        // Avoid "index shifting" issues
        for (let i = fns.length - 1; i >= 0; i--) {
            if (fns[i] === fn) {
                fns.splice(i, 1);
                break;
            }
        }
    }

    #url = undefined;
    #ws = undefined;

    #connected = false;
    #id = undefined;

    #ackId = 0;
    #ackFns = new Map();

    #level = LOG_LEVEL_DEBUG;

    constructor(url) {
        this.#url = url;
        this.connect();
    }

    #onPacket(packet) {
        switch (packet.type) {
            case 'connect':
                this.#onConnect(packet.data.id);
                break;
            case 'disconnect':
                this.#onDisconnect(packet.data.reason);
                break;
            case 'ack':
                if (this.#ackFns.has(packet.data.id)) {
                    const ackFn = this.#ackFns.get(packet.data.id);
                    ackFn(...packet.data.args);
                    this.#ackFns.delete(packet.data.id);
                }
                break;
            case 'event':
                if (packet.data.ackId > 0) {
                    packet.data.args.push((...args) => {
                        this.#emitAck(packet.data.ackId, ...args);
                    });
                }
                this.#emit(packet.data.event, ...packet.data.args);
                break;
            default:
                this.debug('Unknown packet type:', packet);
                break;
        }
    }

    #onConnect(id) {
        this.#connected = true;
        this.#id = id;

        this.#emit('connect', id);
    }

    connect(url) {
        if (this.#connected) {
            return this;
        }

        this.#ws = new WebSocket(this.#url);
        this.#ws.onopen = (evt) => {
            this.debug('ONOPEN HANDLER', evt);
        };
        this.#ws.onerror = (evt) => {
            this.debug('ONERROR HANDLER', evt);
        };
        this.#ws.onclose = (evt) => {
            this.debug('ONCLOSE HANDLER', evt);
            this.#onDisconnect('socket server disconnected');
        };
        this.#ws.onmessage = (evt) => {
            this.debug('ONMESSAGE HANDLER', evt);
            const packet = JSON.parse(evt.data);
            this.#onPacket(packet);
        };
        return this;
    }

    #onDisconnect(reason) {
        if (!this.#connected) {
            return;
        }

        this.#connected = false;
        this.#id = undefined;

        this.#ws.onclose;
        this.#ws.close();
        this.#ws = undefined;

        this.#ackId = 0;
        this.#ackFns.clear();

        this.#emit('disconnect', reason);
    }

    disconnect() {
        if (!this.#connected) {
            return;
        }

        this.#ws.close();
        return this;
    }

    get id() {
        return this.#id;
    }

    get connected() {
        return this.#connected;
    }

    get disconnected() {
        return !this.#connected;
    }

    emit(event, ...args) {
        if (!this.#connected) {
            return;
        }

        const ackFn = args.at(-1);
        const hasAckFn = typeof ackFn === 'function';
        if (hasAckFn) {
            this.#ackId += 1;
            this.#ackFns.set(this.#ackId, ackFn);

            // Remove the "ack" function
            args.pop();
        }

        const packet = JSON.stringify({
            type: 'event',
            data: {
                event,
                args: args,
                ackId: hasAckFn ? this.#ackId : 0,
            },
        });
        this.#ws.send(packet);
        return true;
    }

    #emitAck(id, ...args) {
        const packet = JSON.stringify({
            type: 'ack',
            data: {
                id: id,
                args: args,
            },
        });
        this.#ws.send(packet);
    }

    on(event, fn) {
        this.#on(event, fn);
        return this;
    }

    off(event, fn) {
        this.#off(event, fn);
        return this;
    }

    debug(...args) {
        if (this.#level <= LOG_LEVEL_DEBUG) {
            console.log(...args);
        }
    }

    setLogLevel(level) {
        this.#level = level;
    }
}

export function io(url) {
    const socket = new Socket(url);
    return socket;
}
