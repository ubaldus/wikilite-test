// Copyright (C) 2024-2025 by Ubaldo Porcheddu <ubaldo@eja.it>

let article = null;
let currentSection = 0;
let currentText = 0;
let utterance = null;
let isPlaying = false;
let isLastItem = false;
let isSpeakingSection = false;
let currentSpeakingResultIndex = 0;
let isReadingResults = false;
let isVoiceSearch = false;

const urlParams = new URLSearchParams(window.location.search);
const language = urlParams.get('language') || 'en-US';
const ai = urlParams.get('ai') === 'true';
const speechInput = document.getElementById('speechInput');
const startSpeechButton = document.getElementById('startSpeech');
const searchForm = document.getElementById('searchForm');

async function fetchArticle(articleId) {
    try {
        document.getElementById('loadingSpinner').classList.remove('d-none');
        const response = await fetch(`/api/article?id=${articleId}`);
        const data = await response.json();
        if (data.status === 'success') {
            article = data.article[0];
            displayArticle();
            if (isVoiceSearch) {
                speechSynthesis.cancel();
                isReadingResults = false;
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
    document.getElementById('articleTitle').textContent = article.title;
    document.title = article.title;
    const container = document.getElementById('articleTextContent');
    container.innerHTML = '';
    article.sections.forEach((section, sectionIndex) => {
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
    toggleArrowsVisibility(true);
}

function speakCurrent(section = false) {
    const currentElement = document.getElementById(`text-${currentSection}-${currentText}`);
    const sectionTitle = document.getElementById(`section-${currentSection}`);
    const articleTitle = document.getElementById('articleTitle');

    if (section) {
        if (sectionTitle) {
            speak(sectionTitle.textContent, true);
            highlightCurrent(true);
        }
    } else {
        if (currentSection === 0 && currentText === 0 && !isSpeakingSection) {
            speak(articleTitle.textContent, true);
            highlightCurrent(true);
        } else if (currentElement) {
            speak(currentElement.textContent, false);
            highlightCurrent(false);
        }
    }
}

function speakResult(text, onEnd = null) {
    if (utterance) {
        speechSynthesis.cancel();
    }
    utterance = new SpeechSynthesisUtterance(text);
    utterance.lang = language;
    utterance.onend = () => {
        if (onEnd) onEnd();
    };
    speechSynthesis.speak(utterance);
}

function speak(text, isSectionTitle = false) {
    if (utterance) {
        speechSynthesis.cancel();
    }
    utterance = new SpeechSynthesisUtterance(text);
    utterance.lang = language;
    isSpeakingSection = isSectionTitle;
    if (isSectionTitle) {
        utterance.volume = 1.0;
        utterance.rate = 0.9;
        text += '... ... ...';
    }
    utterance.onend = () => {
        if (isPlaying && !isLastItem) {
            if (isSpeakingSection) {
                setTimeout(() => {
                    currentText = 0;
                    const firstTextElement = document.getElementById(`text-${currentSection}-${currentText}`);
                    if (firstTextElement) {
                        speak(firstTextElement.textContent, false);
                        highlightCurrent(false);
                    }
                }, 2000);
            } else {
                if (currentText === article.sections[currentSection].texts.length - 1) {
                    if (currentSection < article.sections.length - 1) {
                        currentSection++;
                        currentText = 0;
                        const sectionTitle = document.getElementById(`section-${currentSection}`);
                        if (sectionTitle) {
                            speak(sectionTitle.textContent, true);
                            highlightCurrent(true);
                        }
                    } else {
                        isLastItem = true;
                        isPlaying = false;
                        document.getElementById('playPauseIcon').className = 'bi bi-play-fill';
                    }
                } else {
                    nextText();
                }
            }
        }
    };
    speechSynthesis.speak(utterance);
}

function highlightCurrent(section = false) {
    document.querySelectorAll('.highlight, .highlight-section').forEach(el => {
        el.classList.remove('highlight', 'highlight-section');
    });
    if (section) {
        const sectionTitle = document.getElementById(`section-${currentSection}`);
        if (sectionTitle) {
            sectionTitle.classList.add('highlight-section');
            sectionTitle.scrollIntoView({ behavior: 'smooth', block: 'center' });
        }
    } else {
        const currentElement = document.getElementById(`text-${currentSection}-${currentText}`);
        if (currentElement) {
            currentElement.classList.add('highlight');
            currentElement.scrollIntoView({ behavior: 'smooth', block: 'center' });
        }
    }
}

function playPause() {
    isPlaying = !isPlaying;
    isLastItem = false;
    const icon = document.getElementById('playPauseIcon');
    if (isPlaying) {
        icon.className = 'bi bi-pause-fill';
        speakCurrent();
    } else {
        icon.className = 'bi bi-play-fill';
        speechSynthesis.cancel();
    }
}

function nextSection() {
    if (currentSection < article.sections.length - 1) {
        currentSection++;
        currentText = 0;
        isLastItem = false;
        if (isPlaying) {
            speakCurrent(true);
        } else {
            highlightCurrent(true);
        }
    }
}

function prevSection() {
    if (currentSection > 0) {
        currentSection--;
        currentText = 0;
        isLastItem = false;
        if (isPlaying) {
            speakCurrent(true);
        } else {
            highlightCurrent(true);
        }
    }
}

function nextText() {
    if (currentText < article.sections[currentSection].texts.length - 1) {
        currentText++;
        isLastItem = false;
    } else if (currentSection < article.sections.length - 1) {
        currentSection++;
        currentText = 0;
        isLastItem = false;
    } else {
        isLastItem = true;
        isPlaying = false;
        document.getElementById('playPauseIcon').className = 'bi bi-play-fill';
        return;
    }
    if (isPlaying) {
        speakCurrent();
    } else {
        highlightCurrent();
    }
}

function prevText() {
    if (currentText > 0) {
        currentText--;
    } else if (currentSection > 0) {
        currentSection--;
        currentText = article.sections[currentSection].texts.length - 1;
    }
    isLastItem = false;
    if (isPlaying) {
        speakCurrent();
    } else {
        highlightCurrent();
    }
}

function createSearchCheckboxes() {
    const checkboxContainer = document.createElement('div');
    checkboxContainer.className = 'mt-2 text-center';

    const searchTypes = [
        { id: 'titleSearch', label: 'Title', value: 'title', checked: true },
        { id: 'contentSearch', label: 'Lexical', value: 'content' },
    ];

    if (ai) {
        searchTypes.push({ id: 'vectorSearch', label: 'Semantic', value: 'vectors' });
    }

    searchTypes.forEach(type => {
        const formCheck = document.createElement('div');
        formCheck.className = 'form-check form-check-inline';

        const input = document.createElement('input');
        input.className = 'form-check-input';
        input.type = 'checkbox';
        input.id = type.id;
        input.name = 'searchType';
        input.value = type.value;
        if (type.checked) {
            input.checked = true;
        }

        const label = document.createElement('label');
        label.className = 'form-check-label';
        label.htmlFor = type.id;
        label.textContent = type.label;

        formCheck.appendChild(input);
        formCheck.appendChild(label);
        checkboxContainer.appendChild(formCheck);
    });

    searchForm.appendChild(checkboxContainer);
}

function submitSearch(event) {
    if (event) {
        event.preventDefault();
    }

    document.getElementById('articleTextContent').innerHTML = '';
    document.getElementById('articleTitle').textContent = '';
    toggleArrowsVisibility(false);
    document.getElementById('resultsContainer').style.display = 'block';

    const query = document.getElementById('speechInput').value;
    const searchTypes = Array.from(document.querySelectorAll('input[name="searchType"]:checked')).map(el => el.value);

    if (query && searchTypes.length > 0) {
        document.getElementById('resultsContainer').querySelector('ol').innerHTML = '';
        document.getElementById('resultsContainer').querySelector('.alert')?.remove();
        document.getElementById('loadingSpinner').classList.remove('d-none');

        let completedSearches = 0;
        let totalResults = 0;

        searchTypes.forEach(type => {
            performSearch(query, type, (results) => {
                completedSearches++;
                totalResults += results.length;

                if (completedSearches === searchTypes.length) {
                    document.getElementById('loadingSpinner').classList.add('d-none');

                    if (totalResults === 0) {
                        const noResults = document.createElement('div');
                        noResults.className = 'alert alert-info';
                        noResults.textContent = `No results found for "${query}"`;
                        document.getElementById('resultsContainer').appendChild(noResults);
                    } else {
                        currentSpeakingResultIndex = 0;
                        isReadingResults = isVoiceSearch;
                        if (isVoiceSearch) {
                            readNextResultTitle();
                        }
                    }
                }
            });
        });
    }
}

function performSearch(query, type, onComplete) {
    const endpoint = `/api/search/${type}`;
    const payload = { query: query, limit: parseInt(document.getElementById("limit").value) };

    fetch(endpoint, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify(payload),
    })
        .then(response => response.json())
        .then(data => {
            if (data.status === 'success') {
                displayResults(data.results, type);
                onComplete(data.results);
            } else {
                console.error('Error:', data.message);
                onComplete([]);
            }
        })
        .catch(error => {
            console.error('Error:', error);
            onComplete([]);
        });
}

function readNextResultTitle() {
    if (!isReadingResults) return;

    const results = document.querySelectorAll('#resultsContainer ol li');
    if (currentSpeakingResultIndex < results.length) {
        const title = results[currentSpeakingResultIndex].querySelector('a').textContent;
        speakResult(title, () => {
            setTimeout(() => {
                currentSpeakingResultIndex++;
                if (currentSpeakingResultIndex < results.length) {
                    readNextResultTitle();
                } else {
                    currentSpeakingResultIndex = 0;
                    readNextResultTitle();
                }
            }, 2000);
        });
    }
}

function displayResults(results, type) {
    const container = document.getElementById('resultsContainer').querySelector('ol');

    if (results.length > 0) {
        results.forEach((result, index) => {
            const li = document.createElement('li');
            li.className = 'list-group-item d-flex justify-content-between align-items-start';

            const divContent = document.createElement('div');
            divContent.className = 'ms-2 me-auto';

            const titleLink = document.createElement('a');
            titleLink.href = '#';
            titleLink.className = 'text-decoration-none';
            titleLink.textContent = result.title;
            titleLink.addEventListener('click', () => {
                fetchArticle(result.article_id);
                document.getElementById('resultsContainer').classList.add('d-none');
                document.getElementById('articleContent').classList.remove('d-none');
            });

            const text = document.createElement('p');
            text.textContent = result.text;

            divContent.appendChild(titleLink);
            divContent.appendChild(text);

            const typeSup = document.createElement('div');
            typeSup.innerHTML = `<sup>[${type.toUpperCase().substring(0, 1)}]</sup>`;

            li.appendChild(divContent);
            li.appendChild(typeSup);
            container.appendChild(li);
        });
    }
}

function toggleArrowsVisibility(show) {
    const arrowsBlock = document.querySelector('.control-buttons');
    if (arrowsBlock) {
        if (show) {
            arrowsBlock.classList.remove('d-none');
            arrowsBlock.classList.add('d-flex');
        } else {
            arrowsBlock.classList.remove('d-flex');
            arrowsBlock.classList.add('d-none');
        }
    }
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


if ('webkitSpeechRecognition' in window || 'SpeechRecognition' in window) {
    const SpeechRecognition = window.SpeechRecognition || window.webkitSpeechRecognition;
    const recognition = new SpeechRecognition();

    recognition.lang = language;
    recognition.continuous = false;

    recognition.onstart = () => {
        if (startSpeechButton) startSpeechButton.disabled = true;
        if (isReadingResults) {
          speechSynthesis.cancel();
          isReadingResults = false;
        } 
        startSpeechButton.classList.add('blinking');
    };

    recognition.onresult = (event) => {
        const speechResult = event.results[0][0].transcript;
        speechInput.value = speechResult;
        if (startSpeechButton) {
          startSpeechButton.disabled = false;
          startSpeechButton.classList.remove('blinking');
        }
        isVoiceSearch = true;
    };

    recognition.onerror = (event) => {
        if (startSpeechButton) {
          startSpeechButton.disabled = false;
          startSpeechButton.classList.remove('blinking');
        }
        console.error("Speech Recognition Error:", event.error);
        isVoiceSearch = false;
    };

    recognition.onend = () => {
        if (startSpeechButton) {
          startSpeechButton.disabled = false;
          startSpeechButton.classList.remove('blinking');
        }
        submitSearch(null);
    };

    if (startSpeechButton) {
        startSpeechButton.addEventListener('click', () => {
            recognition.start();
        });
    }
} else {
    document.getElementById("startSpeech").style.display = "none";
}

if (startSpeechButton && !/Chrome/.test(navigator.userAgent)) {
    startSpeechButton.style.display = 'none';
}

if (searchForm) {
    createSearchCheckboxes();
    searchForm.addEventListener('submit', function (event) {
        submitSearch(event);
    });
}


document.getElementById('playPause').addEventListener('click', playPause);
document.getElementById('nextText').addEventListener('click', nextText);
document.getElementById('prevText').addEventListener('click', prevText);
document.getElementById('nextSection').addEventListener('click', nextSection);
document.getElementById('prevSection').addEventListener('click', prevSection);
document.addEventListener('keydown', (event) => {
  if (article) {
    switch (event.code) {
        case 'Digit1':
        case 'Home':
        case 'Escape':
          event.preventDefault();
          speechSynthesis.cancel();
          location.reload(true);
          break;
        case 'Digit5':
        case 'Enter':
        case 'Space':
            event.preventDefault();
            playPause();
            break;
        case 'Digit6':
        case 'ArrowRight':
            event.preventDefault();
            nextText();
            break;
        case 'Digit4':
        case 'ArrowLeft':
            event.preventDefault();
            prevText();
            break;
        case 'Digit8':
        case 'ArrowUp':
            event.preventDefault();
            prevSection();
            break;
        case 'Digit2':
        case 'ArrowDown':
            event.preventDefault();
            nextSection();
            break;
      }
  }
  if (isReadingResults) {
    switch(event.code) {
        case 'Enter':
        case 'Space':
                event.preventDefault();
                speechSynthesis.cancel();
                isReadingResults = false;
                const results = document.querySelectorAll('#resultsContainer ol li');
                if (currentSpeakingResultIndex < results.length) {
                    const titleLink = results[currentSpeakingResultIndex].querySelector('a');
                    titleLink.click();
                }
      }
  }
  if (startSpeechButton.style.display !== 'none') {
    switch(event.code) {
      case 'NumLock':
      case 'ControlLeft':
      case 'ControlRight':
        startSpeechButton.click();
        break;
    }
  }
});
