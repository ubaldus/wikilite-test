// Copyright (C) 2024-2025 by Ubaldo Porcheddu <ubaldo@eja.it>

async function fetchArticle(articleId) {
    try {
        document.getElementById('loadingSpinner').classList.remove('d-none');
        const response = await fetch(`/api/article?id=${articleId}`);
        const data = await response.json();
        if (data.status === 'success') {
            App.article = data.article;
            displayArticle();
            if (App.isVoiceSearch) {
                speechSynthesis.cancel();
                App.isReadingResults = false;
                playPause();
            }
        }
    } catch (error) {
        console.error('Error fetching article:', error);
    } finally {
        document.getElementById('loadingSpinner').classList.add('d-none');
    }
}

function displayArticle() {
    document.getElementById('articleTitle').textContent = App.article.title;
    document.title = App.article.title;
    const container = document.getElementById('articleTextContent');
    container.innerHTML = '';
    App.article.sections.forEach((section, sectionIndex) => {
        const sectionDiv = document.createElement('div');
        sectionDiv.className = 'mb-4';
        const title = document.createElement('h2');
        title.textContent = section.title;
        title.id = `section-${sectionIndex}`;
        sectionDiv.appendChild(title);
        section.texts.forEach((text, textIndex) => {
            const p = document.createElement('p');
            p.textContent = text;
            p.id = `text-${sectionIndex}-${textIndex}`;
            sectionDiv.appendChild(p);
        });
        container.appendChild(sectionDiv);
    });
    toggleSearchAndArticleVisibility(true);
    setupArticleCommands();
}

function toggleSearchAndArticleVisibility(showArticle) {
    const searchSection = document.getElementById('searchSection');
    const articleContent = document.getElementById('articleContent');

    if (showArticle) {
        searchSection.classList.add('d-none');
        articleContent.classList.remove('d-none');
    } else {
        searchSection.classList.remove('d-none');
        articleContent.classList.add('d-none');
    }
}

function setupArticleCommands() {
    const recognition = new (window.SpeechRecognition || window.webkitSpeechRecognition)();
    recognition.lang = App.language;
    recognition.continuous = false;

    recognition.onresult = (event) => {
        const command = event.results[0][0].transcript.toLowerCase();
        handleArticleCommand(command);
    };

    recognition.onerror = (event) => {
        console.error("Speech Recognition Error:", event.error);
    };

    recognition.onend = () => {
        if (App.isPlaying) {
					recognition.start();
        }
    };

    document.getElementById('startSpeech').addEventListener('click', () => {
				speakResult(App.sentences[App.language].articlePrompt);
        recognition.start();
    });
}

function playPause() {
    App.isPlaying = !App.isPlaying;
    App.isLastItem = false;
    if (App.isPlaying) {
        speakCurrent();
    } else {
        speechSynthesis.cancel();
    }
}

function nextSection() {
    if (App.currentSection < App.article.sections.length - 1) {
        App.currentSection++;
        App.currentText = 0;
        App.isLastItem = false;
        if (App.isPlaying) {
            speakCurrent(true);
        } else {
            highlightCurrent(true);
        }
    }
}

function prevSection() {
    if (App.currentSection > 0) {
        App.currentSection--;
        App.currentText = 0;
        App.isLastItem = false;
        if (App.isPlaying) {
            speakCurrent(true);
        } else {
            highlightCurrent(true);
        }
    }
}

function nextText() {
    if (App.currentText < App.article.sections[App.currentSection].texts.length - 1) {
        App.currentText++;
        App.isLastItem = false;
    } else if (App.currentSection < App.article.sections.length - 1) {
        App.currentSection++;
        App.currentText = 0;
        App.isLastItem = false;
    } else {
        App.isLastItem = true;
        App.isPlaying = false;
        return;
    }
    if (App.isPlaying) {
        speakCurrent();
    } else {
        highlightCurrent();
    }
}

function prevText() {
    if (App.currentText > 0) {
        App.currentText--;
    } else if (App.currentSection > 0) {
        App.currentSection--;
        App.currentText = App.article.sections[App.currentSection].texts.length - 1;
    }
    App.isLastItem = false;
    if (App.isPlaying) {
        speakCurrent();
    } else {
        highlightCurrent();
    }
}

function speakCurrent(section = false) {
    const currentElement = document.getElementById(`text-${App.currentSection}-${App.currentText}`);
    const sectionTitle = document.getElementById(`section-${App.currentSection}`);
    const articleTitle = document.getElementById('articleTitle');

    if (section) {
        if (sectionTitle) {
            speak(sectionTitle.textContent, true);
            highlightCurrent(true);
        }
    } else {
        if (App.currentSection === 0 && App.currentText === 0 && !App.isSpeakingSection) {
            speak(articleTitle.textContent, true);
            highlightCurrent(true);
        } else if (currentElement) {
            speak(currentElement.textContent, false);
            highlightCurrent(false);
        }
    }
}

function highlightCurrent(section = false) {
    document.querySelectorAll('.highlight, .highlight-section').forEach(el => {
        el.classList.remove('highlight', 'highlight-section');
    });
    if (section) {
        const sectionTitle = document.getElementById(`section-${App.currentSection}`);
        if (sectionTitle) {
            sectionTitle.classList.add('highlight-section');
            sectionTitle.scrollIntoView({ behavior: 'smooth', block: 'center' });
        }
    } else {
        const currentElement = document.getElementById(`text-${App.currentSection}-${App.currentText}`);
        if (currentElement) {
            currentElement.classList.add('highlight');
            currentElement.scrollIntoView({ behavior: 'smooth', block: 'center' });
        }
    }
}

function handleArticleCommand(command) {
		console.log("STT article:", command)
    const commands = App.commands[App.language];
    const helpMessage = App.sentences[App.language].articleHelp;

    switch (command) {
				case commands.articlePlay:
						App.isPlaying = true;
						speakCurrent();
						break
        case commands.articleRepeat:
            speakCurrent();
            break;
				case commands.articleStop:
						App.isPlaying = false;
						speechSynthesis.cancel();
						break;						
        case commands.articleNext:
            nextText();
            break;
        case commands.articlePrevious:
            prevText();
            break;
        case commands.articleNextSection:
            nextSection();
            break;
        case commands.articlePreviousSection:
            prevSection();
            break;
        case commands.articleHome:
            speechSynthesis.cancel();
						document.location.reload();
            break;
        default:
						speak(helpMessage);
            break;
    }
}


