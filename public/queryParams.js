const globalQueryParams = new URLSearchParams(window.location.search);

export function getGlobalQueryParam(key, initialState) {
    const value = globalQueryParams.get(key);
    if (value === null) {
        return initialState;
    }

    if (typeof initialState === 'string') {
        return value;
    }

    try {
        return JSON.parse(value);
    } catch {
        // Ignore error
    }
    return undefined;
}

export function setGlobalQueryParam(key, value, defaultValue) {
    if (value === defaultValue) {
        globalQueryParams.delete(key);
        historyPushState();
        return;
    }

    // Skip, when null or undefined
    if (isNil(value)) {
        return;
    }

    if (typeof value === 'string') {
        globalQueryParams.set(key, value);
    } else {
        globalQueryParams.set(key, JSON.stringify(value));
    }
    historyPushState();
}

function historyPushState() {
    if (globalQueryParams.size > 0) {
        window.history.pushState({}, '', `?${globalQueryParams.toString()}`);
    } else {
        const url = new URL(window.location.href);
        window.history.pushState({}, '', url.pathname);
    }
}

function isNil(obj) {
    return obj === null || obj === 'null' || obj === undefined || obj === 'undefined';
}
