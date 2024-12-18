// Copyright (C) 2024 by Ubaldo Porcheddu <ubaldo@eja.it>

let utterance;
let isSpeaking = false;

const speechInput = document.getElementById('speechInput');
const startSpeechButton = document.getElementById('startSpeech');
const speakButton = document.querySelector('.speak-button');
const speakIcon = document.querySelector('.speak-button i');

if (!'speechSynthesis' in window) { 
	speakButton.style.display = 'none';
}


if ('webkitSpeechRecognition' in window || 'SpeechRecognition' in window) {
	const SpeechRecognition = window.SpeechRecognition || window.webkitSpeechRecognition;
	const recognition = new SpeechRecognition();

	const browserLanguage = navigator.language || navigator.userLanguage;
	recognition.lang = browserLanguage;

	recognition.continuous = false;

	recognition.onstart = () => {
					if(startSpeechButton) startSpeechButton.disabled = true;
						console.log("Speech Recognition started");
	};

	recognition.onresult = (event) => {
					const speechResult = event.results[0][0].transcript;
					speechInput.value = speechResult;
					if(startSpeechButton) startSpeechButton.disabled = false;
	};

	recognition.onerror = (event) => {
					if(startSpeechButton) startSpeechButton.disabled = false;
					console.error("Speech Recognition Error:", event.error);
	};

	recognition.onend = () => {
					if(startSpeechButton) startSpeechButton.disabled = false;
					console.log("Speech Recognition ended");
	};

	if (startSpeechButton) {
					startSpeechButton.addEventListener('click', () => {
													recognition.start();
					});
				}
} else {
				document.getElementById("startSpeech").style.display="none";
}


function speakPageContent() {
				const elementsToSpeak = document.querySelectorAll('h1, h2, h3, h4, h5, h6, p');
				const allText = Array.from(elementsToSpeak)
								.map(el => el.textContent)
								.join(". ");

				utterance = new SpeechSynthesisUtterance(allText);
				utterance.lang = navigator.language || "en-US";
				utterance.rate = 1.0;
				utterance.pitch = 1.0;

				speakIcon.classList.remove('bi-volume-up-fill');
				speakIcon.classList.add('bi-volume-mute-fill');
				isSpeaking = true;


				speechSynthesis.speak(utterance);

				utterance.onend = () => {
								speakIcon.classList.remove('bi-volume-mute-fill');
								speakIcon.classList.add('bi-volume-up-fill');
								isSpeaking = false;

				};
}

speakButton.addEventListener('click', () => {
	if(isSpeaking) {
		speechSynthesis.cancel()
		speakIcon.classList.remove('bi-volume-mute-fill');
		speakIcon.classList.add('bi-volume-up-fill');
		isSpeaking = false;
	} else {
		speakPageContent();
		}
});
