// Copyright (C) 2024-2025 by Ubaldo Porcheddu <ubaldo@eja.it>


async function articleFetch(articleId) {
    try {
        document.getElementById('loadingSpinner').classList.remove('d-none');
        if (App.isVoiceSearch) {
            beepLoadingStop = beepLoading();
        }
        const response = await fetch(`/api/article?id=${articleId}`);
        const data = await response.json();
        if (data.status === 'success') {
            App.article = data.article;
            articleDisplay();
            if (App.isVoiceSearch) {
                articlePlay();
            }
        }
    } catch (error) {
        console.error('Error fetching article:', error);
    } finally {
        if (App.isVoiceSearch) {
            beepLoadingStop();
        }
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

async function articlePlay() {
    App.isPlaying = true;
    for (; App.currentSection < App.article.sections.length; App.currentSection++) {
        if (!App.isPlaying) {
            break;
        }
        await articlePlayCurrent();
    }
    if (App.currentSection > 0 && App.currentSection === App.article.sections.length) {
        App.currentSection--;
    }
}

async function articleStop() {
    App.isPlaying = false;
    await TTSStop();
}

function articlePlayNextSection() {
    if (App.currentSection < App.article.sections.length - 1) {
        App.currentSection++;
    } else {
        App.currentSection = 0;
    }
    articlePlayCurrent();
}

function articlePlayPreviousSection() {
    if (App.currentSection > 0) {
        App.currentSection--;
    } else {
        App.currentSection = App.article.sections.length - 1;
    }
    articlePlayCurrent();
}

async function articlePlayCurrent() {
    if (!App.isPlaying || !App.article.sections[App.currentSection]) {
        return;
    }

    highlightCurrent(true);
    const titleToSpeak = (App.currentSection === 0 && App.article.title) ? App.article.title : App.article.sections[App.currentSection].title;
    await TTS(titleToSpeak);
    if (!App.isPlaying) return;
    await sleep(1500);
    if (!App.isPlaying) return;


    highlightCurrent(false);
    const contentToSpeak = App.article.sections[App.currentSection].content;
    await TTS(contentToSpeak);
}

function highlightCurrent(section = false) {
    document.querySelectorAll('.highlight, .highlight-section').forEach(el => {
        el.classList.remove('highlight', 'highlight-section');
    });

    if (section) {
        let sectionTitle = document.getElementById(`section-${App.currentSection}`);
        if (App.currentSection == 0) {
            sectionTitle = document.getElementById('articleTitle');
        }
        if (sectionTitle) {
            sectionTitle.classList.add('highlight-section');
            sectionTitle.scrollIntoView({
                behavior: 'smooth',
                block: 'center'
            });
        }
    } else {
        const currentElement = document.getElementById(`content-${App.currentSection}`);
        if (currentElement) {
            currentElement.classList.add('highlight');
            currentElement.scrollIntoView({
                behavior: 'smooth',
                block: 'center'
            });
        }
    }
}
