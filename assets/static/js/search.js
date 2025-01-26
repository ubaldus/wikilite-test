// Copyright (C) 2024-2025 by Ubaldo Porcheddu <ubaldo@eja.it>

function createSearchCheckboxes() {
    const checkboxContainer = document.createElement('div');
    checkboxContainer.className = 'mt-2 text-center';

    const searchTypes = [
        { id: 'titleSearch', label: 'Title', value: 'title', checked: false },
        { id: 'lexicalSearch', label: 'Lexical', value: 'lexical', checked: true },
    ];

    if (App.ai) {
        searchTypes.push({ id: 'semanticSearch', label: 'Semantic', value: 'semantic', checked: true });
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

    App.searchForm.appendChild(checkboxContainer);
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
                if (data.results) {
                    displayResults(data.results, type);
                    onComplete(data.results);
                } else {
                    onComplete([]);
                }
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

            li.appendChild(divContent);
            container.appendChild(li);
        });
    }
}

function readNextResultTitle() {
    if (!App.isReadingResults) return;

    const results = document.querySelectorAll('#resultsContainer ol li');
    if (App.currentSpeakingResultIndex < results.length) {
        if (App.currentSpeakingResultIndex > 0) {
            results[App.currentSpeakingResultIndex - 1].classList.remove('highlight');
        }

        const title = results[App.currentSpeakingResultIndex].querySelector('a').textContent;

        speakResult(title, () => {
            results[App.currentSpeakingResultIndex].classList.add('highlight');
            results[App.currentSpeakingResultIndex].scrollIntoView({ behavior: 'smooth', block: 'center' });

                speakResult(App.sentences[App.language].searchOpen, () => {
                    const recognition = new (window.SpeechRecognition || window.webkitSpeechRecognition)();
                    recognition.lang = App.language;
                    recognition.continuous = false;

                    recognition.onresult = (event) => {
                        const response = event.results[0][0].transcript.toLowerCase();
												console.log("STT results:", response)
                        if (response === App.commands[App.language].searchOpen) {
                            results[App.currentSpeakingResultIndex].querySelector('a').click();
                        }
                    };

										recognition.onend = () => {
											App.currentSpeakingResultIndex++;
											if (App.currentSpeakingResultIndex < results.length) {
												readNextResultTitle();
											} else {
												results[App.currentSpeakingResultIndex - 1].classList.remove('highlight');
												App.currentSpeakingResultIndex = 0;
												readNextResultTitle();
											}
										};          

                    recognition.start();
                });
        });
    }
}

function submitSearch(event) {
    if (event) {
        event.preventDefault();
    }

    document.getElementById('articleTextContent').innerHTML = '';
    document.getElementById('articleTitle').textContent = '';
    document.getElementById('resultsContainer').style.display = 'block';

    const query = App.speechInput.value;
    const searchTypes = Array.from(document.querySelectorAll('input[name="searchType"]:checked')).map(el => el.value);

    if (query && searchTypes.length > 0) {
        document.getElementById('resultsContainer').querySelector('ol').innerHTML = '';
        document.getElementById('resultsContainer').querySelector('.alert')?.remove();
        document.getElementById('loadingSpinner').classList.remove('d-none');
        if (App.isVoiceSearch) {
            beepLoadingStart();
        }

        let completedSearches = 0;
        let totalResults = 0;
        let allResults = [];

        searchTypes.forEach(type => {
            performSearch(query, type, (results) => {
                completedSearches++;
                totalResults += results.length;
                allResults = allResults.concat(results);

                if (completedSearches === searchTypes.length) {
                    document.getElementById('loadingSpinner').classList.add('d-none');

                    if (totalResults === 0) {
                        const noResults = document.createElement('div');
                        noResults.className = 'alert alert-info';
                        noResults.textContent = `No results found for "${query}"`;
                        document.getElementById('resultsContainer').appendChild(noResults);
                        if (App.isVoiceSearch) {
                            beepLoadingStop();
                            beepAlert();
                            speakResult(App.sentences[App.language].searchNoResults);
                        }
                    } else if (totalResults === 1) {
                        fetchArticle(allResults[0].article_id);
                        document.getElementById('resultsContainer').classList.add('d-none');
                        document.getElementById('articleContent').classList.remove('d-none');
                        if (App.isVoiceSearch) {
                            beepLoadingStop();
                        }
                    } else {
                        App.currentSpeakingResultIndex = 0;
                        App.isReadingResults = App.isVoiceSearch;
                        if (App.isVoiceSearch) {
                            beepLoadingStop();
                            readNextResultTitle();
                        }
                    }
                }
            });
        });
    }
}
