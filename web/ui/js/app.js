// app.js - Main application entry point

// Application initialization and module management
class BridgoApp {
    constructor() {
        this.authManager = null;
        this.dataSourceManager = null;
        this.virtualViewManager = null;
    }

    init() {
        this.authManager = new AuthManager();
        this.authManager.init();

        if (window.location.pathname.includes('/db_connections') || 
            window.location.pathname.includes('/dashboard')) {
            this.dataSourceManager = new DataSourceManager();
            this.dataSourceManager.init();
        }

        if (window.location.pathname.includes('/virtual_views') ||
            window.location.pathname.includes('/dashboard')) {
            this.virtualViewManager = new VirtualViewManager();
            this.virtualViewManager.init();
        }
    }

    getAuthManager() {
        return this.authManager;
    }

    getDataSourceManager() {
        return this.dataSourceManager;
    }

    getVirtualViewManager() {
        return this.virtualViewManager;
    }
}

let bridgoApp = null;

document.addEventListener('DOMContentLoaded', () => {
    bridgoApp = new BridgoApp();
    bridgoApp.init();
});

if (typeof window !== 'undefined') {
    window.BridgoApp = bridgoApp;
}