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
        section.texts.forEach((text, textIndex) => {
            const p = document.createElement('p');
            p.textContent = text;
            p.id = `text-${sectionIndex}-${textIndex}`;
            sectionDiv.appendChild(p);
        });
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
        await articlePlayCurrent(true);
        for (App.currentText = 0; App.currentText < App.article.sections[App.currentSection].texts.length; App.currentText++) {
            if (!App.isPlaying) {
                break;
            }
            await articlePlayCurrent();
        }
    }
    if (App.currentSection > 0) {
        App.currentSection--
    }
}

async function articleStop() {
    App.isPlaying = false;
    await TTSStop();
}

function articlePlayNextSection() {
    App.currentText = 0;
    if (App.currentSection < App.article.sections.length - 1) {
        App.currentSection++;
    } else {
        App.currentSection = 0;
    }
    articlePlayCurrent(true);
}

function articlePlayPreviousSection() {
    App.currentText = 0;
    if (App.currentSection > 0) {
        App.currentSection--;
    } else {
        App.currentSection = App.article.sections.length - 1;
    }
    articlePlayCurrent(true);
}

function articlePlayNextText() {
    if (App.currentText < App.article.sections[App.currentSection].texts.length - 1) {
        App.currentText++;
    } else if (App.currentSection < App.article.sections.length - 1) {
        App.currentSection++;
        App.currentText = 0;
    } else {
        App.currentSection = 0;
        App.currentText = 0;
    }
    articlePlayCurrent();
}

function articlePlayPreviousText() {
    if (App.currentText > 0) {
        App.currentText--;
    } else if (App.currentSection > 0) {
        App.currentSection--;
        App.currentText = App.article.sections[App.currentSection].texts.length - 1;
    }
    articlePlayCurrent();
}

async function articlePlayCurrent(section = false) {
    if (section) {
        highlightCurrent(true);
        if (App.currentSection == 0) {
            await TTS(App.article.title);
        } else {
            await TTS(App.article.sections[App.currentSection].title);
        }
        await sleep(1500);
    } else {
        highlightCurrent(false);
        await TTS(App.article.sections[App.currentSection].texts[App.currentText]);
    }
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
        const currentElement = document.getElementById(`text-${App.currentSection}-${App.currentText}`);
        if (currentElement) {
            currentElement.classList.add('highlight');
            currentElement.scrollIntoView({
                behavior: 'smooth',
                block: 'center'
            });
        }
    }
}