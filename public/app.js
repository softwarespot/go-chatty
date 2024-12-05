import { getGlobalQueryParam, setGlobalQueryParam } from './queryParams.js';
import { io } from './socket.js';

const state = {
    enablePingLatencyChecker: false,
    socket: undefined,
};

const roomNameEl = document.getElementById('room-name');
const joinBtnEl = document.getElementById('join-btn');
const leaveBtnEl = document.getElementById('leave-btn');
const msgLogEl = document.getElementById('msgs-log');
const msgInputEl = document.getElementById('msg-input');
const msgBtnEl = document.getElementById('msg-btn');

hideElement(leaveBtnEl);

const protocol = location.protocol === 'http:' ? 'ws' : 'wss';
const socket = io(`${protocol}://${location.host}/chat`);
roomNameEl.focus();

const socketId = socket.id;

function pingLatencyChecker() {
    let timerId = 0;
    if (!socket.connected) {
        clearTimeout(timerId);
        return;
    }

    console.log('Sending ping');
    const currMs = Date.now();

    socket.emit('ping', () => {
        const diffMs = Date.now() - currMs;
        console.log(`Received pong: Latency is ${diffMs}ms`);
        logMessage('System', `Latency between the client and server is ${diffMs}ms.`);

        timerId = setTimeout(() => pingLatencyChecker(), 10_000);
    });
}

socket.on('connect', (id) => {
    logMessage('System', `Connected using socket ID ${socket.id}.`);

    if (state.enablePingLatencyChecker) {
        pingLatencyChecker();
    }

    const roomName = getGlobalQueryParam('room', '');
    roomNameEl.value = roomName;

    if (roomName.trim() !== '') {
        // Automatically join the room
        sendJoinRoom();
    }
});

socket.on('disconnect', (reason) => {
    logMessage('System', `Disconnected socket. Reason: ${reason}.`);

    roomNameEl.disabled = false;
    joinBtnEl.disabled = true;
    showElement(joinBtnEl);
    hideElement(leaveBtnEl);
    msgInputEl.disabled = true;
    msgBtnEl.disabled = true;
});

function sendJoinRoom() {
    const roomName = getGlobalQueryParam('room', '');
    if (roomName.trim() === '') {
        alert('Please enter a non-empty room name.');
        return;
    }

    socket.emit('join', roomName, () => {
        roomNameEl.disabled = true;
        hideElement(joinBtnEl);
        showElement(leaveBtnEl);
        msgInputEl.disabled = false;
        msgBtnEl.disabled = false;
        msgInputEl.focus();
    });
}

roomNameEl.addEventListener('keypress', ({ key }) => {
    if (key === 'Enter') {
        sendJoinRoom();
    }
});
roomNameEl.addEventListener('input', () => {
    setGlobalQueryParam('room', roomNameEl.value, '');
});

joinBtnEl.addEventListener('click', () => sendJoinRoom());

leaveBtnEl.addEventListener('click', () => {
    const roomName = getGlobalQueryParam('room', '');
    socket.emit('leave', roomName);

    roomNameEl.disabled = false;
    showElement(joinBtnEl);
    hideElement(leaveBtnEl);
    msgInputEl.disabled = true;
    msgBtnEl.disabled = true;
});

function sendMessage() {
    const msg = msgInputEl.value.trim();
    if (msg === '') {
        alert('Please enter a non-empty message.');
        return;
    }

    msgInputEl.value = '';
    msgInputEl.focus();
    msgInputEl.selectionStart = 0;
    msgInputEl.selectionEnd = 0;
    msgInputEl.setSelectionRange(0, 0);

    socket.emit('message', msg);
}

msgBtnEl.addEventListener('click', () => {
    sendMessage();
});

msgInputEl.addEventListener('keypress', (evt) => {
    if (evt.key === 'Enter') {
        evt.preventDefault();
        sendMessage();
    }
});

function logMessage(sender, msg) {
    const msgEl = document.createElement('p');
    msgEl.classList.add(sender.toLowerCase());

    const headerEl = document.createElement('span');
    headerEl.innerHTML = `<strong>${sender}: </strong>`;
    msgEl.appendChild(headerEl);

    const contentEl = document.createElement('span');
    contentEl.textContent = msg;
    msgEl.appendChild(contentEl);

    msgLogEl.appendChild(msgEl);
    msgLogEl.scrollTop = msgLogEl.scrollHeight;
}

socket.on('message', (sender, msg) => {
    logMessage(sender, msg);
});

function hideElement(el) {
    el.style.display = 'none';
}

function showElement(el) {
    el.style.display = 'initial';
}
