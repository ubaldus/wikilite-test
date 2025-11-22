// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

window.App = {
    article: null,
    searchInput: document.getElementById('searchInput'),
    searchForm: document.getElementById('searchForm'),
    language: new URLSearchParams(window.location.search).get('language') || 'en',
    ai: new URLSearchParams(window.location.search).get('ai') === 'true'
};

if (App.searchForm) {
    if (App.ai) {
        document.getElementById('semanticSearch').checked = true;
    } else {
        document.getElementById('semanticSearchCheck').classList.add('d-none');
    }

    App.searchForm.addEventListener('submit', function(event) {
        submitSearch(event);
    });
}

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

function submitSearch(event) {
    event.preventDefault();

    document.getElementById('articleTextContent').innerHTML = '';
    document.getElementById('articleTitle').textContent = '';
    document.getElementById('resultsContainer').style.display = 'block';

    const query = App.searchInput.value;
    const searchTypes = Array.from(document.querySelectorAll('input[name="searchType"]:checked'))
        .map(el => el.value);

    if (query && searchTypes.length > 0) {
        document.getElementById('resultsContainer').querySelector('ol').innerHTML = '';
        document.getElementById('resultsContainer').querySelector('.alert')?.remove();
        document.getElementById('loadingSpinner').classList.remove('d-none');

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
                    } else if (totalResults === 1) {
                        articleFetch(allResults[0].article_id);
                        document.getElementById('resultsContainer').classList.add('d-none');
                        document.getElementById('articleContent').classList.remove('d-none');
                    }
                }
            });
        });
    }
}

async function articleFetch(articleId) {
    try {
        document.getElementById('loadingSpinner').classList.remove('d-none');
        const response = await fetch(`/api/article?id=${articleId}`);
        const data = await response.json();
        
        if (data.status === 'success') {
            App.article = data.article;
            articleDisplay();
        }
    } catch (error) {
        console.error('Error fetching article:', error);
    } finally {
        document.getElementById('loadingSpinner').classList.add('d-none');
    }
}

function articleDisplay() {
    App.isArticle = true;
    document.title = App.article.title;
    document.getElementById('articleTitle').textContent = App.article.title;
    
    const container = document.getElementById('articleTextContent');
    container.innerHTML = '';
    
    App.article.sections.forEach((section, sectionIndex) => {
        const sectionDiv = document.createElement('div');
        sectionDiv.className = 'mb-4';
        
        const title = document.createElement('h2');
        title.textContent = section.title;
        title.id = `section-${sectionIndex}`;
        sectionDiv.appendChild(title);

        const p = document.createElement('p');
        p.textContent = section.content;
        p.id = `content-${sectionIndex}`;
        sectionDiv.appendChild(p);

        container.appendChild(sectionDiv);
    });
    
    document.getElementById('searchSection').classList.add('d-none');
    document.getElementById('articleContent').classList.remove('d-none');
}
