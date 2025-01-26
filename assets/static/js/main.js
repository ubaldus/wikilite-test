// Copyright (C) 2024-2025 by Ubaldo Porcheddu <ubaldo@eja.it>

if (App.searchForm) {
    createSearchCheckboxes();
    App.searchForm.addEventListener('submit', function (event) {
        submitSearch(event);
    });
}
