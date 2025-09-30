// ui.js - Handles UI updates and event delegation for the wiki frontend
export class WikiUI {
    constructor(app) {
        this.app = app;
        this.initEventDelegation();
    }

    initEventDelegation() {
        // Event delegation for page list
        document.getElementById('pageList').addEventListener('click', (e) => {
            const link = e.target.closest('a');
            if (link) {
                e.preventDefault();
                this.app.loadPage(link.textContent);
            }
        });
        // Event delegation for search results
        document.getElementById('searchResults').addEventListener('click', (e) => {
            const result = e.target.closest('a'); // Changed from .search-result to a
            if (result) {
                e.preventDefault();
                this.app.loadPage(result.textContent);
            }
        });
    }
}
