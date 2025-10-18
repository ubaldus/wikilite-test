// Copyright (C) 2024-2025 by Ubaldo Porcheddu <ubaldo@eja.it>


window.App = {
    article: null,
    currentSection: 0,
    currentText: 0,
    currentSpeakingResultIndex: 0,
    isPlaying: false,
    isReadingResults: false,
    isVoiceSearch: false,
    isArticle: false,
    isSearchResults: false,
    beepLoadingStop: null,
    searchInput: document.getElementById('searchInput'),
    searchForm: document.getElementById('searchForm'),
    speechButton: document.getElementById('speechButton'),
    language: new URLSearchParams(window.location.search).get('language') || 'en',
    ai: new URLSearchParams(window.location.search).get('ai') === 'true',
    locale: {
        'en': {
            'sentences': {
                searchPrompt: "What do you want to search?",
                searching: "Searching...",
                searchNoResults: "I am sorry, I didn't find anything.",
                searchOpen: "Do you want to open this?",
                articleHelp: "You can say: play, stop, repeat, next, previous or home.",
                articlePrompt: "How can I help you?"
            },
            'commands': {
                searchOpen: ["yes"],
                searchBack: ["back"],
                articlePlay: ["play", "continue"],
                articleStop: ["stop"],
                articleRepeat: ["repeat"],
                articleNext: ["next"],
                articlePrevious: ["previous"],
                articleHome: "home"
            },
        },
        'it': {
            'sentences': {
                searchPrompt: "Cosa vorresti cercare?",
                searching: "sto cercando...",
                searchNoResults: "Mi dispiace ma non ho trovato niente.",
                searchOpen: "Vuoi lèggere quest'articolo?",
                articleHelp: "Puoi dire: leggi, pausa, ripeti, indietro, avanti o ricarica.",
                articlePrompt: "Dimmi?"
            },
            'commands': {
                searchOpen: ["sì", "si", "leggi", "leggilo", "sì leggilo", "sì leggi"],
                searchBack: ["indietro"],
                articlePlay: ["leggi", "continua"],
                articleStop: ["stop", "pausa"],
                articleRepeat: ["ripeti"],
                articleNext: ["avanti"],
                articlePrevious: ["indietro"],
                articleHome: ["ricarica"]
            },
        },
        'es': {
            'sentences': {
                searchPrompt: "¿Qué quieres buscar?",
                searching: "Buscando...",
                searchNoResults: "Lo siento, no he encontrado nada.",
                searchOpen: "¿Quieres abrir esto?",
                articleHelp: "Puedes decir: reproducir, parar, repetir, siguiente, anterior o inicio.",
                articlePrompt: "¿Cómo puedo ayudarte?"
            },
            'commands': {
                searchOpen: ["sí", "abrir"],
                searchBack: ["atrás"],
                articlePlay: ["reproducir", "continuar"],
                articleStop: ["parar", "detener"],
                articleRepeat: ["repetir"],
                articleNext: ["siguiente"],
                articlePrevious: ["anterior"],
                articleHome: ["inicio"]
            },
        },
        'de': {
            'sentences': {
                searchPrompt: "Was möchten Sie suchen?",
                searching: "Suche läuft...",
                searchNoResults: "Es tut mir leid, ich habe nichts gefunden.",
                searchOpen: "Möchten Sie das öffnen?",
                articleHelp: "Sie können sagen: abspielen, stoppen, wiederholen, nächster, vorheriger oder Startseite.",
                articlePrompt: "Wie kann ich Ihnen helfen?"
            },
            'commands': {
                searchOpen: ["ja", "öffnen"],
                searchBack: ["zurück"],
                articlePlay: ["abspielen", "weiter"],
                articleStop: ["stopp", "anhalten"],
                articleRepeat: ["wiederholen"],
                articleNext: ["nächster"],
                articlePrevious: ["vorheriger"],
                articleHome: ["Startseite"]
            },
        },
    }
};
