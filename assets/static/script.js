// Copyright (C) 2024 by Ubaldo Porcheddu <ubaldo@eja.it>


function createSearchCheckboxes() {
    const checkboxContainer = document.createElement('div');
    checkboxContainer.className = 'mt-2';

    const searchTypes = [
        { id: 'titleSearch', label: 'Title', value: 'title', checked: true },
        { id: 'contentSearch', label: 'Content', value: 'content' },
        { id: 'vectorSearch', label: 'Vector', value: 'vectors' }
    ];

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
    const query = document.getElementById('speechInput').value;
    const searchTypes = Array.from(document.querySelectorAll('input[name="searchType"]:checked')).map(el => el.value);

    if (query && searchTypes.length > 0) {
        document.getElementById('resultsContainer').querySelector('ol').innerHTML = '';
        document.getElementById('resultsContainer').querySelector('.alert')?.remove();

        document.getElementById('loadingSpinner').style.display = 'block';

        let completedSearches = 0;
        let totalResults = 0;

        searchTypes.forEach(type => {
            performSearch(query, type, (results) => {
                completedSearches++;
                totalResults += results.length;

                if (completedSearches === searchTypes.length) {
                    document.getElementById('loadingSpinner').style.display = 'none';

                    if (totalResults === 0) {
                        const noResults = document.createElement('div');
                        noResults.className = 'alert alert-info';
                        noResults.textContent = `No results found for "${query}"`;
                        document.getElementById('resultsContainer').appendChild(noResults);
                    }
                }
            });
        });
    }
}

function performSearch(query, type, onComplete) {
    const endpoint = `/api/search/${type}`;
    const payload = { query: query, limit: 3 };

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

function displayResults(results, type) {
    const container = document.getElementById('resultsContainer').querySelector('ol');

    if (results.length > 0) {
        results.forEach(result => {
            const li = document.createElement('li');
            li.className = 'list-group-item d-flex justify-content-between align-items-start';

            const divContent = document.createElement('div');
            divContent.className = 'ms-2 me-auto';

            const titleLink = document.createElement('a');
						if ('speechSynthesis' in window) {
							titleLink.href= `/static/tts.html?id=${result.article_id}&locale=${language}`;
						} else {
							titleLink.href = `article?id=${result.article_id}`;
						}
            titleLink.className = 'text-decoration-none';
            titleLink.textContent = result.title;

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

const speechInput = document.getElementById('speechInput');
const startSpeechButton = document.getElementById('startSpeech');
const searchForm = document.getElementById('searchForm');

if (startSpeechButton && !/Chrome/.test(navigator.userAgent)) {
    startSpeechButton.style.display = 'none';
}

if ('webkitSpeechRecognition' in window || 'SpeechRecognition' in window) {
    const SpeechRecognition = window.SpeechRecognition || window.webkitSpeechRecognition;
    const recognition = new SpeechRecognition();

    recognition.lang = language;
    recognition.continuous = false;

    recognition.onstart = () => {
        if (startSpeechButton) startSpeechButton.disabled = true;
        console.log("Speech Recognition started");
    };

    recognition.onresult = (event) => {
        const speechResult = event.results[0][0].transcript;
        speechInput.value = speechResult;
        if (startSpeechButton) startSpeechButton.disabled = false;
    };

    recognition.onerror = (event) => {
        if (startSpeechButton) startSpeechButton.disabled = false;
        console.error("Speech Recognition Error:", event.error);
    };

    recognition.onend = () => {
        if (startSpeechButton) startSpeechButton.disabled = false;
        console.log("Speech Recognition ended");
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

if (searchForm) {
    createSearchCheckboxes();
    searchForm.addEventListener('submit', function (event) {
        submitSearch(event);
    });
}
