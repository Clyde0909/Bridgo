// auth.js - Authentication functionality

class AuthManager {
    constructor() {
        this.registerForm = null;
        this.loginForm = null;
        this.messageElement = null;
        this.logoutButton = null;
    }

    init() {
        this.registerForm = document.getElementById('registerForm');
        this.loginForm = document.getElementById('loginForm');
        this.messageElement = document.getElementById('message');
        this.logoutButton = document.getElementById('logoutButton');

        this.setupEventListeners();
        this.handleAuthRedirects();
    }

    setupEventListeners() {
        if (this.registerForm) {
            this.registerForm.addEventListener('submit', (e) => this.handleRegister(e));
        }

        if (this.loginForm) {
            this.setupLoginForm();
            this.loginForm.addEventListener('submit', (e) => this.handleLogin(e));
        }

        if (this.logoutButton) {
            this.logoutButton.addEventListener('click', (e) => {
                e.preventDefault();
                logout();
            });
        }
    }

    setupLoginForm() {
        // Check for username in query params and pre-fill
        const urlParams = new URLSearchParams(window.location.search);
        const usernameFromQuery = urlParams.get('username');
        if (usernameFromQuery) {
            this.loginForm.username.value = decodeURIComponent(usernameFromQuery);
        }
    }

    async handleRegister(e) {
        e.preventDefault();
        const username = this.registerForm.username.value;
        const email = this.registerForm.email.value;
        const password = this.registerForm.password.value;
        
        displayMessage(this.messageElement, '');

        try {
            const response = await fetch('/api/register', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ username, email, password }),
            });

            const result = await response.json();

            if (response.ok) {
                displayMessage(this.messageElement, 'Registration successful! Redirecting to login page.', 'success');
                setTimeout(() => {
                    window.location.href = `/login?username=${encodeURIComponent(username)}`;
                }, 2000);
            } else {
                displayMessage(this.messageElement, `Error: ${result.message || response.statusText}`, 'error');
            }
        } catch (error) {
            displayMessage(this.messageElement, `Unexpected error occurred: ${error.message}`, 'error');
        }
    }

    async handleLogin(e) {
        e.preventDefault();
        const username = this.loginForm.username.value;
        const password = this.loginForm.password.value;
        
        displayMessage(this.messageElement, '');

        try {
            const response = await fetch('/api/login', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ username, password }),
            });

            const result = await response.json();

            if (response.ok) {
                setAuthToken(result.token);
                displayMessage(this.messageElement, 'Login successful! Redirecting to dashboard.', 'success');
                setTimeout(() => {
                    window.location.href = '/dashboard';
                }, 1000);
            } else {
                displayMessage(this.messageElement, `Error: ${result.message || response.statusText}`, 'error');
            }
        } catch (error) {
            displayMessage(this.messageElement, `Unexpected error occurred: ${error.message}`, 'error');
        }
    }

    handleAuthRedirects() {
        // If on a page that requires auth, check immediately
        if (window.location.pathname.includes('/dashboard')) {
            requireAuth();
        }

        // Optional: Redirect if already logged in and visiting login/register
        if (isAuthenticated()) {
            if (window.location.pathname.includes('/login') || window.location.pathname.includes('/register')) {
                // Uncomment to enable auto-redirect
                // window.location.href = '/dashboard';
            }
        }
    }
}

// Export for module use
if (typeof module !== 'undefined' && module.exports) {
    module.exports = AuthManager;
}
