// Copyright (C) 2024-2025 by Ubaldo Porcheddu <ubaldo@eja.it>


let audioContext = null;

function initAudioContext() {
    if (!audioContext) {
        audioContext = new(window.AudioContext || window.webkitAudioContext)();
    }
    if (audioContext.state === 'suspended') {
        audioContext.resume();
    }
    return audioContext;
}

function beep({
    frequency = 440,
    type = 'sine',
    duration = 200,
    volume = 0.5,
    repeat = false,
    interval = 1000
}) {
    initAudioContext();
    let timeoutId = null;

    function playBeep(time) {
        const oscillator = audioContext.createOscillator();
        const gainNode = audioContext.createGain();

        oscillator.type = type;
        oscillator.frequency.setValueAtTime(frequency, time);
        gainNode.gain.setValueAtTime(volume, time);

        oscillator.connect(gainNode);
        gainNode.connect(audioContext.destination);

        oscillator.start(time);
        gainNode.gain.exponentialRampToValueAtTime(0.01, time + duration / 1000);
        oscillator.stop(time + duration / 1000);
    }

    let time = audioContext.currentTime;
    playBeep(time);

    if (repeat) {
        function scheduleBeep() {
            time += interval / 1000;
            playBeep(time);
            timeoutId = setTimeout(scheduleBeep, interval);
        }
        timeoutId = setTimeout(scheduleBeep, interval);

        return () => {
            if (timeoutId) {
                clearTimeout(timeoutId);
                timeoutId = null;
            }
        };
    }
}

async function TTS(text, lang = App.language) {
    return new Promise((resolve) => {
        if (window.speechSynthesis) {
            speechSynthesis.cancel();
            const utterance = new SpeechSynthesisUtterance(text);
            utterance.lang = lang;
            utterance.onend = () => resolve();
            utterance.onerror = () => resolve();
            speechSynthesis.speak(utterance);
        } else {
            resolve();
        }
    });
}

async function TTSStop() {
    if (window.speechSynthesis) {
        await speechSynthesis.cancel();
    }
}

async function STT(lang = App.language) {
    await TTSStop();
    return new Promise((resolve) => {
        if (!('webkitSpeechRecognition' in window) && !('SpeechRecognition' in window)) {
            resolve('');
            return;
        }

        const SpeechRecognition = window.SpeechRecognition || window.webkitSpeechRecognition;
        const recognition = new SpeechRecognition();
        recognition.lang = lang;
        recognition.onresult = (event) => {
            const transcript = event.results[0][0].transcript.toLowerCase();
            console.log("STT", transcript);
            resolve(transcript);
        };

        recognition.onerror = () => {
            resolve('');
        };

        recognition.onend = () => {
            beepStop();
            App.speechButton.disabled = false;
            resolve('');
        };

        App.speechButton.disabled = true;
        beepStart();
        recognition.start();
    });
}

function beepStart() {
    return beep({
        frequency: 880,
        duration: 100,
        volume: 0.3
    });
}

function beepStop() {
    return beep({
        frequency: 660,
        duration: 300,
        volume: 0.3
    });
}

function beepLoading() {
    return beep({
        frequency: 440,
        duration: 50,
        volume: 0.2,
        repeat: true,
        interval: 1000
    });
}

function beepAlert() {
    return beep({
        frequency: 310,
        type: 'square',
        duration: 500,
        volume: 0.25
    });
}

function sleep(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
}