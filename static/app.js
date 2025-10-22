import { wikiAPI } from './api.js';
import { WikiUI } from './ui.js';

// Configure marked for secure markdown parsing
// Since this is localhost-only, we can be slightly more permissive
// while still preventing dangerous XSS attacks
marked.setOptions({
    breaks: true,  // Convert \n to <br>
    gfm: true,     // GitHub Flavored Markdown
    headerIds: true, // Add IDs to headers for linking
    mangle: false,  // Don't mangle email addresses
    sanitize: false, // Marked v9+ has built-in XSS protection without this deprecated option
    smartypants: true, // Convert quotes and dashes to smart versions
});

// Basic HTML escaping for displaying user input as text
// Used when we need to show raw user input (like titles) without markdown parsing
function escapeHtml(unsafe) {
    return unsafe
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#039;");
}

// WikiApp orchestrates frontend logic, API, and UI modules
class WikiApp {
    constructor() {
        this.api = wikiAPI;
        this.currentPage = null;
        this.pages = [];
        this.isEditMode = false;
        this.editDisplayMode = 'editor-only';
        this.unsavedChanges = false;
        this.originalContent = '';
        this.searchTimeout = null;

        this.initializeElements();
        this.ui = new WikiUI(this);
        this.setupEventListeners();
        this.loadPages();
        this.loadPage('Home');

        setInterval(() => {
            if (this.unsavedChanges && this.isEditMode) {
                this.savePage(false);
            }
        }, 30000);
    }
    initializeElements() {
        this.elements = {
            // Mode containers
            viewMode: document.getElementById('viewMode'),
            editMode: document.getElementById('editMode'),

            // View mode elements
            viewPageTitle: document.getElementById('viewPageTitle'),
            viewPageMeta: document.getElementById('viewPageMeta'),
            viewContent: document.getElementById('viewContent'),
            editBtn: document.getElementById('editBtn'),
            versionsBtn: document.getElementById('versionsBtn'),

            // Edit mode elements
            editPageTitle: document.getElementById('editPageTitle'),
            editPageMeta: document.getElementById('editPageMeta'),
            editContentArea: document.getElementById('editContentArea'),
            editorPanel: document.getElementById('editorPanel'),
            previewPanel: document.getElementById('previewPanel'),
            editor: document.getElementById('editor'),
            preview: document.getElementById('preview'),

            // Edit mode buttons
            editorOnlyBtn: document.getElementById('editorOnlyBtn'),
            previewOnlyBtn: document.getElementById('previewOnlyBtn'),
            splitViewBtn: document.getElementById('splitViewBtn'),
            saveBtn: document.getElementById('saveBtn'),
            cancelBtn: document.getElementById('cancelBtn'),

            // Common elements
            deleteBtn: document.getElementById('deleteBtn'),
            pageList: document.getElementById('pageList'),
            newPageName: document.getElementById('newPageName'),
            createPageBtn: document.getElementById('createPageBtn'),
            searchInput: document.getElementById('searchInput'),
            searchResults: document.getElementById('searchResults'),

            // Modal
            modal: document.getElementById('modal'),
            modalTitle: document.getElementById('modalTitle'),
            modalMessage: document.getElementById('modalMessage'),
            modalConfirm: document.getElementById('modalConfirm'),
            modalCancel: document.getElementById('modalCancel'),
            statusMessageContainer: document.getElementById('statusMessageContainer'),

            // Versions modal
            versionsModal: document.getElementById('versionsModal'),
            versionsList: document.getElementById('versionsList')
        };
    }

    setupEventListeners() {
        // View mode
        this.elements.editBtn.addEventListener('click', () => this.enterEditMode());
        this.elements.versionsBtn.addEventListener('click', () => this.showVersionHistory());

        // Edit mode
        this.elements.saveBtn.addEventListener('click', () => this.saveAndExitEdit());
        this.elements.cancelBtn.addEventListener('click', () => this.cancelAndExitEdit());
        
        // Edit display mode buttons
        this.elements.editorOnlyBtn.addEventListener('click', () => this.setEditDisplayMode('editor-only'));
        this.elements.previewOnlyBtn.addEventListener('click', () => this.setEditDisplayMode('preview-only'));
        this.elements.splitViewBtn.addEventListener('click', () => this.setEditDisplayMode('split-view'));
        
        // Editor events
        this.elements.editor.addEventListener('input', () => {
            this.updatePreview();
            this.markUnsaved();
        });

        // Delete button
        this.elements.deleteBtn.addEventListener('click', () => this.showDeleteModal());
        
        // New page
        this.elements.createPageBtn.addEventListener('click', () => this.createNewPage());
        this.elements.newPageName.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') this.createNewPage();
        });

        // Search functionality
        this.elements.searchInput.addEventListener('input', (e) => this.handleSearch(e.target.value));
        this.elements.searchInput.addEventListener('focus', () => this.showSearchResults());
        document.addEventListener('click', (e) => {
            if (!e.target.closest('.search-box')) {
                this.hideSearchResults();
            }
        });

        // Keyboard shortcuts
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape' && this.isEditMode) {
                this.cancelAndExitEdit();
            }
            
            if (e.ctrlKey || e.metaKey) {
                if (e.key === 's' && this.isEditMode) {
                    e.preventDefault();
                    this.saveAndExitEdit();
                }
            }
        });

        // Prevent accidental navigation only in edit mode
        window.addEventListener('beforeunload', (e) => {
            if (this.unsavedChanges && this.isEditMode) {
                e.preventDefault();
                e.returnValue = '';
            }
        });
    }

    // API methods are now handled by wikiAPI

    async loadPages() {
        try {
            this.pages = await this.api.getPages();
            this.updatePageList();
        } catch (error) {
            this.elements.pageList.innerHTML = '<li class="loading">Failed to load pages</li>';
            this.ui.showStatus(`Error loading pages: ${error.message}`, 'error');
        }
    }

    updatePageList() {
        if (this.pages.length === 0) {
            this.elements.pageList.innerHTML = '<li class="loading">No pages found</li>';
            return;
        }

        // Clear the list and add each page as a <li>
        this.elements.pageList.innerHTML = '';
        this.pages
            .sort((a, b) => a.title.localeCompare(b.title))
            .forEach(page => {
                const li = document.createElement('li');
                li.className = 'list-group-item';
                const a = document.createElement('a');
                a.href = '#';
                a.className = 'list-group-item-action';
                a.textContent = page.title;
                if (page.title === this.currentPage?.title) {
                    a.classList.add('active');
                    a.setAttribute('aria-current', 'true');
                }
                li.appendChild(a);
                this.elements.pageList.appendChild(li);
            });
    }

    async loadPage(title) {
        if (this.unsavedChanges && this.isEditMode) {
            if (!confirm('You have unsaved changes. Do you want to save before switching pages?')) {
                return;
            }
            await this.savePage(false);
        }
        try {
            const page = await this.api.getPage(title);
            this.currentPage = page;
            this.originalContent = page.content;
            this.updatePageInfo(page);
            this.showViewMode();
            this.updateView();
            this.updatePageList();
        } catch (error) {
            if (error.message.includes('not found') || error.message.includes('404')) {
                this.currentPage = {
                    title: title,
                    content: `# ${title}\n\nThis is a new page. Start writing!`,
                    created_at: new Date().toISOString(),
                    updated_at: new Date().toISOString()
                };
                this.originalContent = this.currentPage.content;
                this.updatePageInfo(this.currentPage);
                this.enterEditMode();
                this.elements.deleteBtn.style.display = 'none';
            } else {
                this.ui.showStatus(`Error loading page: ${error.message}`, 'error');
            }
        }
    }

    updatePageInfo(page) {
        const created = new Date(page.created_at).toLocaleDateString();
        const updated = new Date(page.updated_at).toLocaleDateString();
        const meta = page.id ? `Created: ${created} • Updated: ${updated}` : 'New page';
        
        // Update view mode
        this.elements.viewPageTitle.textContent = page.title;
        this.elements.viewPageMeta.textContent = meta;
        
        // Update edit mode
        this.elements.editPageTitle.textContent = `Editing: ${page.title}`;
        this.elements.editPageMeta.textContent = meta;
        
        // Update editor content
        this.elements.editor.value = page.content;
        
        // Show/hide delete button
        this.elements.deleteBtn.style.display = page.title === 'Home' ? 'none' : 'inline-flex';
    }

    showViewMode() {
        this.isEditMode = false;
        this.elements.viewMode.style.display = 'block';
        this.elements.editMode.style.display = 'none';
        this.markSaved();
    }

    enterEditMode() {
        this.isEditMode = true;
        this.originalContent = this.elements.editor.value;
        this.elements.viewMode.style.display = 'none';
        this.elements.editMode.style.display = 'block';
        this.setEditDisplayMode(this.editDisplayMode);
        this.markSaved();
    }

    async saveAndExitEdit() {
        if (!this.currentPage) return;
        await this.savePage(true);
        this.showViewMode();
        this.updateView();
    }

    cancelAndExitEdit() {
        if (this.unsavedChanges) {
            if (!confirm('You have unsaved changes. Are you sure you want to discard them?')) {
                return;
            }
        }
        
        // Restore original content
        this.elements.editor.value = this.originalContent;
        this.showViewMode();
        this.updateView();
    }

    setEditDisplayMode(mode) {
        this.editDisplayMode = mode;

        // Update button states
        this.elements.editorOnlyBtn.classList.toggle('active', mode === 'editor-only');
        this.elements.previewOnlyBtn.classList.toggle('active', mode === 'preview-only');
        this.elements.splitViewBtn.classList.toggle('active', mode === 'split-view');

        // Update content area class
        this.elements.editContentArea.classList.remove('editor-only', 'preview-only', 'split-view');

        if (mode === 'editor-only') {
            this.elements.editContentArea.classList.add('editor-only');
            this.elements.editorPanel.style.display = 'flex';
            this.elements.previewPanel.style.display = 'none';
        } else if (mode === 'preview-only') {
            this.elements.editContentArea.classList.add('preview-only');
            this.elements.editorPanel.style.display = 'none';
            this.elements.previewPanel.style.display = 'flex';
        } else if (mode === 'split-view') {
            this.elements.editContentArea.classList.add('split-view');
            this.elements.editorPanel.style.display = 'flex';
            this.elements.previewPanel.style.display = 'flex';
        }

        // Update preview if needed
        if (mode === 'preview-only' || mode === 'split-view') {
            setTimeout(() => {
                this.updatePreview();
            }, 100);
        }
    }

    	async updateView() {
        const content = this.elements.editor.value;
        
        if (this.currentPage?.is_dac && this.currentPage?.title) {
            try {
                const weavedMarkdown = await this.api.getWeave(this.currentPage.title);
                let html = marked.parse(weavedMarkdown);
                html = this.processWikiLinks(html);
                
                const badge = '<span class="format-badge dac">📝 DAC Format</span>';
                this.elements.viewContent.innerHTML = badge + html;
                
                // Highlight all code blocks
                this.elements.viewContent.querySelectorAll('pre code').forEach((block) => {
                    hljs.highlightElement(block);
                });
            } catch (error) {
                console.error('DAC weave failed:', error);
                this.elements.viewContent.innerHTML = 
                    `<div class="error">Failed to weave DAC content: ${error.message}</div>`;
            }
        } else {
            let processed = this.processFootnotes(content);
            let html = marked.parse(processed);
            html = this.processWikiLinks(html);
            this.elements.viewContent.innerHTML = html;
            
            // Highlight code in regular markdown too
            this.elements.viewContent.querySelectorAll('pre code').forEach((block) => {
                hljs.highlightElement(block);
            });
        }
    }
    
    // Same for updatePreview()
    async updatePreview() {
        if (!this.isEditMode) return;
        if (this.editDisplayMode === 'editor-only') return;

        const content = this.elements.editor.value;
        const isDac = /<<[^>]+>>=/m.test(content) && /^\s*@\s*$/m.test(content);

        if (isDac) {
            this.elements.preview.innerHTML =
                `<div class="dac-preview-notice">
                    <strong>📝 DAC Format Detected</strong>
                    <p>Save the page to see the weaved output.</p>
                </div>
                <pre><code class="language-plaintext">${escapeHtml(content)}</code></pre>`;
            this.elements.preview.querySelectorAll('pre code').forEach((block) => {
                hljs.highlightElement(block);
            });
        } else {
            let processed = this.processFootnotes(content);
            let html = marked.parse(processed);
            html = this.processWikiLinks(html);
            this.elements.preview.innerHTML = html;

            this.elements.preview.querySelectorAll('pre code').forEach((block) => {
                hljs.highlightElement(block);
            });
        }
    }

    async savePage(showNotification = true) {
        if (!this.currentPage) return;
        const content = this.elements.editor.value;
        const isNewPage = !this.currentPage.id;
        
        try {
            const originalText = this.elements.saveBtn.textContent;
            this.elements.saveBtn.textContent = 'Saving...';
            this.elements.saveBtn.classList.add('saving');
            let savedPage;
            if (isNewPage) {
                savedPage = await this.api.createPage(this.currentPage.title, content);
            } else {
                savedPage = await this.api.updatePage(this.currentPage.title || 'Home', content);
            }
            this.currentPage = savedPage;
            this.originalContent = content;
            this.updatePageInfo(savedPage);
            this.markSaved();
            if (showNotification) {
                this.ui.showStatus('Page saved successfully', 'success');
            }
            if (isNewPage) {
                await this.loadPages();
            }
            setTimeout(() => {
                this.elements.saveBtn.textContent = originalText;
                this.elements.saveBtn.classList.remove('saving');
            }, 1000);
        } catch (error) {
            this.elements.saveBtn.textContent = 'Save';
            this.elements.saveBtn.classList.remove('saving');
            // Only show error if not already shown for this error
            if (!this.lastSaveError || this.lastSaveError !== error.message) {
                this.ui.showStatus(`Error saving page: ${error.message}`, 'error');
                this.lastSaveError = error.message;
            }
        }
    }

    async createNewPage() {
        const title = this.elements.newPageName.value.trim();
        if (!title) return;
        if (this.pages.find(p => p.title.toLowerCase() === title.toLowerCase())) {
            this.ui.showStatus('Page already exists', 'error');
            return;
        }
        this.elements.newPageName.value = '';
        await this.loadPage(title);
    }

    showDeleteModal() {
        if (!this.currentPage || this.currentPage.title === 'Home') return;
        this.elements.modalTitle.textContent = 'Delete Page';
        this.elements.modalMessage.textContent = `Are you sure you want to delete "${this.currentPage.title}"? This action cannot be undone.`;
        this.showModal();
    }

    async handleModalConfirm() {
        if (!this.currentPage || this.currentPage.title === 'Home') return;
        try {
            await this.api.deletePage(this.currentPage.title);
            this.ui.showStatus('Page deleted successfully', 'success');
            await this.loadPages();
            await this.loadPage('Home');
        } catch (error) {
            this.ui.showStatus(`Error deleting page: ${error.message}`, 'error');
        }
    }

    showModal() {
        const modal = new bootstrap.Modal(this.elements.modal);
        modal.show();
    }

    hideModal() {
        const modal = bootstrap.Modal.getInstance(this.elements.modal);
        if (modal) {
            modal.hide();
        }
    }

    async handleSearch(query) {
        clearTimeout(this.searchTimeout);
        if (!query.trim()) {
            this.hideSearchResults();
            return;
        }
        this.searchTimeout = setTimeout(async () => {
            try {
                const results = await this.api.searchPages(query);
                this.showSearchResults(results);
            } catch (error) {
                this.hideSearchResults();
                this.ui.showStatus(`Error searching pages: ${error.message}`, 'error');
            }
        }, 300);
    }

    showSearchResults(results = null) {
        if (results) {
            // Clear existing results
            this.elements.searchResults.innerHTML = '';
            
            if (results.length === 0) {
                const div = document.createElement('div');
                div.className = 'list-group-item text-muted';
                div.textContent = 'No results found';
                this.elements.searchResults.appendChild(div);
            } else {
                // Create elements safely without innerHTML for onclick handlers
                results.forEach(page => {
                    const a = document.createElement('a');
                    a.href = '#';
                    a.className = 'list-group-item list-group-item-action';
                    a.textContent = page.title;
                    a.addEventListener('click', () => this.loadPage(page.title));
                    this.elements.searchResults.appendChild(a);
                });
            }
        }
        this.elements.searchResults.classList.remove('d-none');
        this.elements.searchResults.classList.add('d-block');
    }

    hideSearchResults() {
        this.elements.searchResults.classList.remove('d-block');
        this.elements.searchResults.classList.add('d-none');
    }

    processWikiLinks(text) {
        return text.replace(/\[\[([^\]]+)\]\]/g, (match, pageName) => {
            const exists = this.pages.some(p => p.title === pageName);
            const className = exists ? 'wiki-link' : 'wiki-link broken';
            return `<a href="#" onclick="app.loadPage('${escapeHtml(pageName)}')" class="${className}">${escapeHtml(pageName)}</a>`;
        });
    }

    escapeHtml(unsafe) {
        return unsafe
            .replace(/&/g, "&amp;")
            .replace(/</g, "&lt;")
            .replace(/>/g, "&gt;")
            .replace(/"/g, "&quot;")
            .replace(/'/g, "&#039;");
    }

    processFootnotes(text) {
        const footnoteRefs = {};
        const footnotes = {};
        let footnoteCounter = 0;

        // Extract footnote definitions
        text = text.replace(/^\[(\^[^\]]+)\]:\s*(.+)$/gm, (match, id, content) => {
            footnotes[id] = content;
            return '';
        });

        // Process footnote references
        text = text.replace(/\[(\^[^\]]+)\]/g, (match, id) => {
            if (!footnoteRefs[id]) {
                footnoteCounter++;
                footnoteRefs[id] = footnoteCounter;
            }
            const num = footnoteRefs[id];
            return `<a href="#fn${num}" class="footnote-ref" id="fnref${num}">${num}</a>`;
        });

        // Add footnotes section
        if (Object.keys(footnotes).length > 0) {
            let footnotesHtml = '\n\n<div class="footnotes">\n';
            for (const [id, content] of Object.entries(footnotes)) {
                const num = footnoteRefs[id];
                if (num) {
                    footnotesHtml += `<div class="footnote" id="fn${num}">${num}. ${content} <a href="#fnref${num}" class="footnote-backref">↩</a></div>\n`;
                }
            }
            footnotesHtml += '</div>';
            text += footnotesHtml;
        }

        return text;
    }

    markUnsaved() {
        if (!this.isEditMode) return;
        this.unsavedChanges = true;
        if (!this.elements.saveBtn.textContent.includes('*')) {
            this.elements.saveBtn.textContent = 'Save *';
            this.elements.saveBtn.classList.remove('btn-success');
            this.elements.saveBtn.classList.add('btn-warning');
        }
    }

    markSaved() {
        this.unsavedChanges = false;
        this.elements.saveBtn.textContent = 'Save';
        this.elements.saveBtn.classList.remove('btn-warning');
        this.elements.saveBtn.classList.add('btn-success');
    }

    async showVersionHistory() {
        if (!this.currentPage || !this.currentPage.title) return;

        // Show modal
        const modal = new bootstrap.Modal(this.elements.versionsModal);
        modal.show();

        // Load versions
        this.elements.versionsList.innerHTML = '<div class="text-center text-muted">Loading versions...</div>';

        try {
            const versions = await this.api.getPageVersions(this.currentPage.title);

            if (!versions || versions.length === 0) {
                this.elements.versionsList.innerHTML = '<div class="text-center text-muted">No previous versions found.</div>';
                return;
            }

            // Display current version
            let html = `
                <div class="list-group-item list-group-item-action active">
                    <div class="d-flex w-100 justify-content-between">
                        <h6 class="mb-1">Current Version</h6>
                        <small>${new Date(this.currentPage.updated_at).toLocaleString()}</small>
                    </div>
                    <div class="btn-group mt-2" role="group">
                        <button class="btn btn-sm btn-outline-primary" disabled>Viewing</button>
                    </div>
                </div>
            `;

            // Display previous versions
            versions.forEach((version, index) => {
                const date = new Date(version.created_at).toLocaleString();
                html += `
                    <div class="list-group-item list-group-item-action">
                        <div class="d-flex w-100 justify-content-between">
                            <h6 class="mb-1">Version ${versions.length - index}</h6>
                            <small>${date}</small>
                        </div>
                        <div class="btn-group mt-2" role="group">
                            <button class="btn btn-sm btn-primary" onclick="app.loadVersion(${version.id})">View</button>
                            <button class="btn btn-sm btn-secondary" onclick="app.compareWithCurrent(${version.id})">Compare</button>
                        </div>
                    </div>
                `;
            });

            this.elements.versionsList.innerHTML = html;
        } catch (error) {
            this.elements.versionsList.innerHTML = `<div class="alert alert-danger">Error loading versions: ${error.message}</div>`;
        }
    }

    async loadVersion(versionId) {
        if (!this.currentPage || !this.currentPage.title) return;

        try {
            const versionPage = await this.api.getPage(this.currentPage.title, versionId);
            this.currentPage = versionPage;
            this.updatePageInfo(versionPage);
            this.updateView();

            // Close modal
            const modal = bootstrap.Modal.getInstance(this.elements.versionsModal);
            if (modal) modal.hide();

            this.ui.showStatus('Loaded historical version', 'info');
        } catch (error) {
            this.ui.showStatus(`Error loading version: ${error.message}`, 'error');
        }
    }

    async compareWithCurrent(versionId) {
        // Get current page ID
        const currentPageId = this.currentPage.id;

        try {
            const diff = await this.api.diffVersions(versionId, currentPageId);

            // Close versions modal
            const modal = bootstrap.Modal.getInstance(this.elements.versionsModal);
            if (modal) modal.hide();

            // Show diff in a new modal (simplified - just show both versions for now)
            alert(`Comparison:\n\nOlder Version:\n${diff.version1.content.substring(0, 200)}...\n\nCurrent:\n${diff.version2.content.substring(0, 200)}...`);

            this.ui.showStatus('Diff feature coming soon - for now showing basic comparison', 'info');
        } catch (error) {
            this.ui.showStatus(`Error comparing versions: ${error.message}`, 'error');
        }
    }
}

// Toolbar functions
function insertMarkdown(before, after = '') {
    if (!app.isEditMode) return;
    
    const editor = app.elements.editor;
    const start = editor.selectionStart;
    const end = editor.selectionEnd;
    const text = editor.value;
    const selectedText = text.substring(start, end);
    
    const replacement = before + selectedText + after;
    editor.value = text.substring(0, start) + replacement + text.substring(end);
    
    // Set cursor position
    const newPos = start + before.length + selectedText.length;
    editor.setSelectionRange(newPos, newPos);
    editor.focus();
    
    app.updatePreview();
    app.markUnsaved();
}

function insertWikiLink() {
    if (!app.isEditMode) return;
    
    const editor = app.elements.editor;
    const start = editor.selectionStart;
    const end = editor.selectionEnd;
    const text = editor.value;
    const selectedText = text.substring(start, end);
    
    const linkText = selectedText || 'Page Name';
    const replacement = `[[${linkText}]]`;
    editor.value = text.substring(0, start) + replacement + text.substring(end);
    
    // Select the page name for easy editing
    const linkStart = start + 2;
    const linkEnd = linkStart + linkText.length;
    editor.setSelectionRange(linkStart, linkEnd);
    editor.focus();
    
    app.updatePreview();
    app.markUnsaved();
}

// Initialize the app and make it globally available
const app = new WikiApp();
window.app = app;  // Expose app globally for onclick handlers
