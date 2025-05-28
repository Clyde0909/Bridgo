// utils.js - Common utility functions

const TOKEN_KEY = 'authToken';

// Function to check if the user is authenticated
function isAuthenticated() {
    return localStorage.getItem(TOKEN_KEY) !== null;
}

// Function to redirect to login if not authenticated
function requireAuth() {
    if (!isAuthenticated()) {
        window.location.href = '/login';
    }
}

// Function to handle logout
function logout() {
    localStorage.removeItem(TOKEN_KEY);
    window.location.href = '/login';
}

// Function to get auth token
function getAuthToken() {
    return localStorage.getItem(TOKEN_KEY);
}

// Function to set auth token
function setAuthToken(token) {
    localStorage.setItem(TOKEN_KEY, token);
}

// Function to display messages with color coding
function displayMessage(element, message, type = 'info') {
    if (!element) return;
    
    element.textContent = message;
    switch (type) {
        case 'success':
            element.style.color = 'green';
            break;
        case 'error':
            element.style.color = 'red';
            break;
        case 'warning':
            element.style.color = 'orange';
            break;
        case 'info':
        default:
            element.style.color = 'blue';
            break;
    }
}

// Function to clear element content
function clearElement(element) {
    if (element) {
        element.innerHTML = '';
    }
}

// Function to create styled div element
function createStyledDiv(styles = {}) {
    const div = document.createElement('div');
    Object.assign(div.style, styles);
    return div;
}

// Export functions for module use
if (typeof module !== 'undefined' && module.exports) {
    module.exports = {
        TOKEN_KEY,
        isAuthenticated,
        requireAuth,
        logout,
        getAuthToken,
        setAuthToken,
        displayMessage,
        clearElement,
        createStyledDiv
    };
}
