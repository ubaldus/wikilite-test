// Copyright (C) 2024-2025 by Ubaldo Porcheddu <ubaldo@eja.it>


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

if (App.speechButton) {
    if (!/Chrome/.test(navigator.userAgent)) {
        App.speechButton.style.display = 'none';
    } else {
        document.getElementById('speechButton').addEventListener('click', async () => {
            if (App.isArticle) {
                const commands = App.locale[App.language].commands;
                await articleStop();
                await TTS(App.locale[App.language].sentences.articlePrompt);
                let command = await STT();

                if (commands.articlePlay.includes(command)) {
                    articlePlay();
                } else if (commands.articleRepeat.includes(command)) {
                    articlePlayCurrent();
                } else if (commands.articleStop.includes(command)) {
                    articleStop();
                } else if (commands.articleNext.includes(command)) {
                    articlePlayNextSection();
                } else if (commands.articlePrevious.includes(command)) {
                    articlePlayPreviousSectiont();
                } else if (commands.articleHome.includes(command)) {
                    document.location.reload();
                } else {
                    await TTS(App.locale[App.language].sentences.articleHelp);
                }
            }
            if (!App.isSearchResults && !App.isArticle) {
                App.isVoiceSearch = false;
                await TTS(App.locale[App.language].sentences.searchPrompt);
                App.searchInput.value = await STT();
                if (App.searchInput.value != "") {
                    App.isVoiceSearch = true;
                    submitSearch();
                }
            }
        });
    }
}
