// Copyright (C) 2024-2025 by Ubaldo Porcheddu <ubaldo@eja.it>


function performSearch(query, type, onComplete) {
    const endpoint = `/api/search/${type}`;
    const payload = {
        query: query,
        limit: parseInt(document.getElementById("limit").value)
    };

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
        App.isSearchResults = true;
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
                articleFetch(result.article_id);
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

async function readNextResultTitle() {
    if (!App.isReadingResults) return;

    const results = document.querySelectorAll('#resultsContainer ol li');
    if (App.currentSpeakingResultIndex < results.length) {
        if (App.currentSpeakingResultIndex > 0) {
            results[App.currentSpeakingResultIndex - 1].classList.remove('highlight');
        }

        results[App.currentSpeakingResultIndex].classList.add('highlight');
        const title = results[App.currentSpeakingResultIndex].querySelector('a').textContent;
        await TTS(title);
        await TTS(App.locale[App.language].sentences.searchOpen)

        let command = await STT();
        if (App.locale[App.language].commands.searchOpen.includes(command)) {
            results[App.currentSpeakingResultIndex].querySelector('a').click();
        } else if (App.locale[App.language].commands.searchBack.includes(command)) {
            results[App.currentSpeakingResultIndex].classList.remove('highlight');
            if (App.currentSpeakingResultIndex > 0) {
                App.currentSpeakingResultIndex--;
            } else {
                App.currentSpeakingResultIndex = results.length - 1;
            }
            readNextResultTitle();
        } else {
            App.currentSpeakingResultIndex++;
            if (App.currentSpeakingResultIndex < results.length) {
                readNextResultTitle();
            } else {
                results[App.currentSpeakingResultIndex - 1].classList.remove('highlight');
                App.currentSpeakingResultIndex = 0;
                readNextResultTitle();
            }
        }
    }
}

function submitSearch(event) {
    if (event) {
        event.preventDefault();
    }

    document.getElementById('articleTextContent').innerHTML = '';
    document.getElementById('articleTitle').textContent = '';
    document.getElementById('resultsContainer').style.display = 'block';

    const query = App.searchInput.value;
    const searchTypes = Array.from(document.querySelectorAll('input[name="searchType"]:checked')).map(el => el.value);

    if (query && searchTypes.length > 0) {
        document.getElementById('resultsContainer').querySelector('ol').innerHTML = '';
        document.getElementById('resultsContainer').querySelector('.alert')?.remove();
        document.getElementById('loadingSpinner').classList.remove('d-none');
        if (App.isVoiceSearch) {
            App.beepLoadingStop = beepLoading();
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
                    if (App.isVoiceSearch) {
                        App.beepLoadingStop();
                    }

                    if (totalResults === 0) {
                        const noResults = document.createElement('div');
                        noResults.className = 'alert alert-info';
                        noResults.textContent = `No results found for "${query}"`;
                        document.getElementById('resultsContainer').appendChild(noResults);
                        if (App.isVoiceSearch) {
                            App.isVoiceSearch = false;
                            beepAlert();
                            TTS(App.locale[App.language].sentences.searchNoResults, App.language);

                        }
                    } else if (totalResults === 1) {
                        articleFetch(allResults[0].article_id);
                        document.getElementById('resultsContainer').classList.add('d-none');
                        document.getElementById('articleContent').classList.remove('d-none');
                    } else {
                        App.currentSpeakingResultIndex = 0;
                        App.isReadingResults = App.isVoiceSearch;
                        if (App.isVoiceSearch) {
                            readNextResultTitle();
                        }
                    }
                }
            });
        });
    }
}
