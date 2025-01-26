// Copyright (C) 2024-2025 by Ubaldo Porcheddu <ubaldo@eja.it>

window.App = {
    article: null,
    currentSection: 0,
    currentText: 0,
    isPlaying: false,
    isLastItem: false,
    isSpeakingSection: false,
    isReadingResults: false,
    isVoiceSearch: false,
    language: new URLSearchParams(window.location.search).get('language') || 'en',
    ai: new URLSearchParams(window.location.search).get('ai') === 'true',
    utterance: null,
    beepAudioContext: null,
    beepLoadingOscillator: null,
    beepLoadingGainNode: null,
    beepTimeoutId: null,
    currentSpeakingResultIndex: 0,
    speechInput: document.getElementById('speechInput'),
    startSpeechButton: document.getElementById('startSpeech'),
    searchForm: document.getElementById('searchForm'),
    sentences: {
        'en': {
            searchPrompt: "What do you want to search?",
            searching: "Searching...",
            searchNoResults: "I am sorry, I didn't find anything.",
            searchOpen: "Do you want to open this?",
            articleHelp: "You can say: play, stop, repeat, next, previous, next section, previous section, or home.",
						articlePrompt: "How can I help you?"
        },
        'it': {
            searchPrompt: "Cosa vorresti cercare?",
            searching: "sto cercando...",
            searchNoResults: "Mi dispiace ma non ho trovato niente.",
            searchOpen: "Vuoi lèggere quest'articolo?",
            articleHelp: "Puoi dire: leggi, pausa, ripeti, indietro, avanti, sezione successiva, sezione precedente o ricarica.",
						articlePrompt: "Dimmi?"
        },
     },
    commands: {
        'en': {
						searchOpen: "yes",
						articlePlay: "play",
						articleStop: "stop",
            articleRepeat: "repeat",
            articleNext: "next",
            articlePrevious: "previous",
            articleNextSection: "next section",
            articlePreviousSection: "previous section",
            articleHome: "home"
        },
        'it': {
						searchOpen: "sì",
						articlePlay: "leggi",
						articleStop: "pausa",
            articleRepeat: "ripeti",
            articleNext: "avanti",
            articlePrevious: "indietro",
            articleNextSection: "sezione successiva",
            articlePreviousSection: "sezione precedente",
            articleHome: "ricarica"
        },
     }
};
