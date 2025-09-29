// api.js - Handles all API requests for the wiki frontend
class WikiAPI {
    constructor(base = '/api') {
        this.base = base;
    }

    async request(endpoint, options = {}) {
        try {
            const response = await fetch(`${this.base}${endpoint}`, {
                headers: {
                    'Content-Type': 'application/json',
                    ...options.headers
                },
                ...options
            });
            if (!response.ok) {
                const errorText = await response.text();
                let errorMessage;
                try {
                    const errorJson = JSON.parse(errorText);
                    errorMessage = errorJson.error || errorText;
                } catch {
                    errorMessage = errorText;
                }
                
                // Provide user-friendly error messages
                if (response.status === 404) {
                    throw new Error('Page not found');
                } else if (response.status === 409) {
                    throw new Error('A page with this title already exists');
                } else if (response.status === 500) {
                    throw new Error('Server error. Please try again later.');
                } else {
                    throw new Error(errorMessage || `Error: ${response.status}`);
                }
            }
            return response.status === 204 ? null : await response.json();
        } catch (error) {
            if (error.name === 'NetworkError' || error.message === 'Failed to fetch') {
                throw new Error('Unable to connect to server. Please check your connection.');
            }
            throw error;
        }
    }

    getPages() { return this.request('/pages'); }
    getPage(title) { return this.request(`/pages/${encodeURIComponent(title)}`); }
    createPage(title, content) {
        return this.request('/pages', {
            method: 'POST',
            body: JSON.stringify({ title, content })
        });
    }
    updatePage(title, content) {
        return this.request(`/pages/${encodeURIComponent(title)}`, {
            method: 'PUT',
            body: JSON.stringify({ title, content })
        });
    }
    deletePage(title) {
        return this.request(`/pages/${encodeURIComponent(title)}`, {
            method: 'DELETE'
        });
    }
    searchPages(query) {
        return this.request(`/pages/search?q=${encodeURIComponent(query)}`);
    }
	  getRaw(title) {
			return fetch(`${this.base}/pages/${encodeURIComponent(title)}/raw`)
				.then(response => response.text())
				.catch(error => {
					if (error.name === 'NetworkError' || error.message === 'Failed to fetch') {
						throw new Error('Unable to connect to server. Please check your connection.');
					}
					throw error;
				});
		}	
	  getWeave(title) {
			return fetch(`${this.base}/pages/${encodeURIComponent(title)}/weave`)
				.then(response => response.text())
				.catch(error => {
					if (error.name === 'NetworkError' || error.message === 'Failed to fetch') {
						throw new Error('Unable to connect to server. Please check your connection.');
					}
					throw error;
				});
		}
	  getTangle(title, chunk) {
			return fetch(`${this.base}/pages/${encodeURIComponent(title)}/tangle/${chunk}`)
				.then(response => response.text())
				.catch(error => {
					if (error.name === 'NetworkError' || error.message === 'Failed to fetch') {
						throw new Error('Unable to connect to server. Please check your connection.');
					}
					throw error;
				});
		}

}

export const wikiAPI = new WikiAPI();
