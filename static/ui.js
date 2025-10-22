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
            const result = e.target.closest('a');
            if (result) {
                e.preventDefault();
                this.app.loadPage(result.textContent);
            }
        });
    }

    showStatus(message, type = 'success') {
        const alertPlaceholder = document.getElementById('statusMessageContainer');
        if (!alertPlaceholder) {
            const body = document.querySelector('body');
            const div = document.createElement('div');
            div.id = 'statusMessageContainer';
            div.style.position = 'fixed';
            div.style.top = '20px';
            div.style.right = '20px';
            div.style.zIndex = '1050';
            body.appendChild(div);
            alertPlaceholder = div;
        }

        const wrapper = document.createElement('div');
        wrapper.innerHTML = [
            `<div class="alert alert-${type} alert-dismissible fade show" role="alert">`,
            `   <div>${message}</div>`,
            '   <button type="button" class="btn-close" data-bs-dismiss="alert" aria-label="Close"></button>',
            '</div>'
        ].join('');

        alertPlaceholder.append(wrapper);

        setTimeout(() => {
            bootstrap.Alert.getInstance(wrapper.querySelector('.alert'))?.close();
        }, 5000);
    }
}
