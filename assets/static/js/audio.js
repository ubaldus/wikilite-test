// Copyright (C) 2024-2025 by Ubaldo Porcheddu <ubaldo@eja.it>

function speakResult(text, onEnd = null) {
    if (App.utterance) {
        speechSynthesis.cancel();
    }
    App.utterance = new SpeechSynthesisUtterance(text);
    App.utterance.lang = App.language;
    App.utterance.onend = () => {
        if (onEnd) onEnd();
    };
    speechSynthesis.speak(App.utterance);
}

function speak(text, isSectionTitle = false) {
    if (App.utterance) {
        speechSynthesis.cancel();
    }
    App.utterance = new SpeechSynthesisUtterance(text);
    App.utterance.lang = App.language;
    App.isSpeakingSection = isSectionTitle;
    if (isSectionTitle) {
        App.utterance.volume = 1.0;
        App.utterance.rate = 0.9;
        text += '... ... ...';
    }
    App.utterance.onend = () => {
        if (App.isPlaying && !App.isLastItem) {
            if (App.isSpeakingSection) {
                setTimeout(() => {
                    App.currentText = 0;
                    const firstTextElement = document.getElementById(`text-${App.currentSection}-${App.currentText}`);
                    if (firstTextElement) {
                        speak(firstTextElement.textContent, false);
                        highlightCurrent(false);
                    }
                }, 2000);
            } else {
                if (App.currentText === App.article.sections[App.currentSection].texts.length - 1) {
                    if (App.currentSection < App.article.sections.length - 1) {
                        App.currentSection++;
                        App.currentText = 0;
                        const sectionTitle = document.getElementById(`section-${App.currentSection}`);
                        if (sectionTitle) {
                            speak(sectionTitle.textContent, true);
                            highlightCurrent(true);
                        }
                    } else {
                        App.isLastItem = true;
                        App.isPlaying = false;
                    }
                } else {
                    nextText();
                }
            }
        }
    };
    speechSynthesis.speak(App.utterance);
}

function beepInitializeContext() {
    if (!App.beepAudioContext) {
        App.beepAudioContext = new (window.AudioContext || window.webkitAudioContext)();
    }
    if (App.beepAudioContext.state === 'suspended') {
        App.beepAudioContext.resume();
    }
}

function beepAlert() {
    beepInitializeContext();

    const errorOscillator = App.beepAudioContext.createOscillator();
    const errorGainNode = App.beepAudioContext.createGain();

    errorOscillator.type = 'square';
    errorOscillator.frequency.setValueAtTime(310, App.beepAudioContext.currentTime);
    errorGainNode.gain.setValueAtTime(0.25, App.beepAudioContext.currentTime);

    errorOscillator.connect(errorGainNode);
    errorGainNode.connect(App.beepAudioContext.destination);

    errorOscillator.start();
    errorOscillator.stop(App.beepAudioContext.currentTime + 0.5);
}

function beepLoadingStart() {
    beepInitializeContext();

    function playBeep(time) {
        App.beepLoadingOscillator = App.beepAudioContext.createOscillator();
        App.beepLoadingGainNode = App.beepAudioContext.createGain();
        App.beepLoadingOscillator.type = 'sine';
        App.beepLoadingOscillator.frequency.setValueAtTime(1000, time);
        App.beepLoadingGainNode.gain.setValueAtTime(0, time);
        App.beepLoadingOscillator.connect(App.beepLoadingGainNode);
        App.beepLoadingGainNode.connect(App.beepAudioContext.destination);
        App.beepLoadingOscillator.start(time);
        App.beepLoadingGainNode.gain.setValueAtTime(1, time);
        App.beepLoadingGainNode.gain.exponentialRampToValueAtTime(0.01, time + 0.05);
        App.beepLoadingGainNode.gain.setValueAtTime(0, time + 0.06);
        App.beepLoadingOscillator.stop(time + 0.06);
    }

    let time = App.beepAudioContext.currentTime;

    function scheduleBeep() {
        playBeep(time);
        time += 1.0;
        App.beepTimeoutId = setTimeout(scheduleBeep, (time - App.beepAudioContext.currentTime) * 1000);
    }

    scheduleBeep();
}

function beepLoadingStop() {
    if (App.beepTimeoutId) {
        clearTimeout(App.beepTimeoutId);
        App.beepTimeoutId = null;
    }
    if (App.beepLoadingOscillator) {
        App.beepLoadingOscillator.stop();
        App.beepLoadingOscillator.disconnect();
        App.beepLoadingOscillator = null;
    }
    if (App.beepLoadingGainNode) {
        App.beepLoadingGainNode.disconnect();
        App.beepLoadingGainNode = null;
    }
}

if ('webkitSpeechRecognition' in window || 'SpeechRecognition' in window) {
    const SpeechRecognition = window.SpeechRecognition || window.webkitSpeechRecognition;
    const recognition = new SpeechRecognition();

    recognition.lang = App.language;
    recognition.continuous = false;

    recognition.onstart = () => {
        if (App.startSpeechButton) App.startSpeechButton.disabled = true;
        if (App.isReadingResults) {
            speechSynthesis.cancel();
            App.isReadingResults = false;
        }
        App.startSpeechButton.classList.add('blinking');
    };

    recognition.onresult = (event) => {
        const speechResult = event.results[0][0].transcript;
				console.log("STT search:", speechResult)
        App.speechInput.value = speechResult;
        if (App.startSpeechButton) {
            App.startSpeechButton.disabled = false;
            App.startSpeechButton.classList.remove('blinking');
        }
        App.isVoiceSearch = true;
        speakResult(App.sentences[App.language].searching, () => {
            submitSearch(null);
        });
    };

    recognition.onerror = (event) => {
        if (App.startSpeechButton) {
            App.startSpeechButton.disabled = false;
            App.startSpeechButton.classList.remove('blinking');
        }
        console.error("Speech Recognition Error:", event.error);
        App.isVoiceSearch = false;
    };

    recognition.onend = () => {
        if (App.startSpeechButton) {
            App.startSpeechButton.disabled = false;
            App.startSpeechButton.classList.remove('blinking');
        }
    };

    if (App.startSpeechButton) {
        App.startSpeechButton.addEventListener('click', () => {
            speakResult(App.sentences[App.language].searchPrompt, () => {
                recognition.start();
            });
        });
    }
} else {
    document.getElementById("startSpeech").style.display = "none";
}

if (App.startSpeechButton && !/Chrome/.test(navigator.userAgent)) {
    App.startSpeechButton.style.display = 'none';
}
