// app.js
console.log("UI JavaScript loaded.");

const TOKEN_KEY = 'authToken';

// Function to check if the user is authenticated
function isAuthenticated() {
    return localStorage.getItem(TOKEN_KEY) !== null;
}

// Function to redirect to login if not authenticated
function requireAuth() {
    if (!isAuthenticated()) {
        window.location.href = '/login'; // Changed from /login.html
    }
}

// Function to handle logout
function logout() {
    localStorage.removeItem(TOKEN_KEY);
    window.location.href = '/login'; // Changed from /login.html
}

document.addEventListener('DOMContentLoaded', () => {
    const registerForm = document.getElementById('registerForm');
    const loginForm = document.getElementById('loginForm');
    const messageElement = document.getElementById('message');
    const logoutButton = document.getElementById('logoutButton'); // For dashboard logout

    // If on a page that requires auth, check immediately
    // This is a simple check; more robust checks might be needed depending on page structure
    if (window.location.pathname.includes('/dashboard')) { // Changed from dashboard.html
        requireAuth();
    }

    if (registerForm) {
        registerForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            const username = registerForm.username.value;
            const email = registerForm.email.value;
            const password = registerForm.password.value;
            messageElement.textContent = ''; // Clear previous messages

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
                    // Redirect to login page with username pre-filled
                    window.location.href = `/login?username=${encodeURIComponent(username)}`;
                } else {
                    messageElement.textContent = 'Error: ' + (result.message || response.statusText);
                    messageElement.style.color = 'red';
                }
            } catch (error) {
                messageElement.textContent = 'An unexpected error occurred: ' + error.message;
                messageElement.style.color = 'red';
            }
        });
    }

    if (loginForm) {
        // Check for username in query params and pre-fill
        const urlParams = new URLSearchParams(window.location.search);
        const usernameFromQuery = urlParams.get('username');
        if (usernameFromQuery) {
            loginForm.username.value = decodeURIComponent(usernameFromQuery);
        }

        loginForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            const username = loginForm.username.value;
            const password = loginForm.password.value;
            messageElement.textContent = ''; // Clear previous messages

            try {
                const response = await fetch('/api/login', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({ username, password }),
                });
                const result = await response.json();
                if (response.ok && result.token) {
                    localStorage.setItem(TOKEN_KEY, result.token);
                    messageElement.textContent = 'Login successful! Redirecting...';
                    messageElement.style.color = 'green';
                    window.location.href = '/dashboard'; // Changed from /dashboard.html
                } else {
                    messageElement.textContent = 'Error: ' + (result.message || 'Login failed. Check credentials.');
                    messageElement.style.color = 'red';
                }
            } catch (error) {
                messageElement.textContent = 'An unexpected error occurred: ' + error.message;
                messageElement.style.color = 'red';
            }
        });
    }

    if (logoutButton) {
        logoutButton.addEventListener('click', (e) => {
            e.preventDefault();
            logout();
        });
    }

    // Optional: Redirect if already logged in and visiting login/register
    if (isAuthenticated()) {
        if (window.location.pathname.includes('/login') || window.location.pathname.includes('/register')) { // Changed from .html
            // window.location.href = '/dashboard'; // Uncomment to enable auto-redirect, changed from /dashboard.html
        }
    }

    // DB Connections Page Logic
    const dbConnectionForm = document.getElementById('dbConnectionForm');
    const connectionMessage = document.getElementById('connectionMessage');
    // const schemaOutput = document.getElementById('schemaOutput'); // Replaced by schemaDisplayArea
    const schemaOutputLoadingStatus = document.getElementById('schemaOutputLoadingStatus');
    const selectableSchemaContainer = document.getElementById('selectableSchemaContainer');
    const virtualViewCreationArea = document.getElementById('virtualViewCreationArea');
    const virtualViewForm = document.getElementById('virtualViewForm');
    const virtualViewMessage = document.getElementById('virtualViewMessage');

    let currentSchemaData = null; // To store the fetched schema for virtual view creation

    if (dbConnectionForm) {
        dbConnectionForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            connectionMessage.textContent = '';
            selectableSchemaContainer.innerHTML = ''; // Clear previous schema
            virtualViewCreationArea.style.display = 'none'; // Hide virtual view form
            virtualViewMessage.textContent = '';
            schemaOutputLoadingStatus.textContent = '스키마 정보를 가져오는 중...';
            currentSchemaData = null;

            const formData = new FormData(dbConnectionForm);
            const data = Object.fromEntries(formData.entries());
            if (data.dbPort) {
                data.dbPort = parseInt(data.dbPort, 10);
            }

            try {
                const token = localStorage.getItem(TOKEN_KEY);
                if (!token) {
                    connectionMessage.textContent = '오류: 인증 토큰을 찾을 수 없습니다. 다시 로그인해주세요.';
                    connectionMessage.style.color = 'red';
                    schemaOutputLoadingStatus.textContent = '스키마 정보를 가져올 수 없습니다.';
                    return;
                }

                const response = await fetch('/api/db/connect-and-fetch-schema', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': `Bearer ${token}`
                    },
                    body: JSON.stringify(data),
                });

                const result = await response.json();

                if (response.ok) {
                    connectionMessage.textContent = '스키마를 성공적으로 가져왔습니다. 데이터 소스 정보가 저장되었습니다.';
                    connectionMessage.style.color = 'green';
                    schemaOutputLoadingStatus.textContent = '아래에서 테이블과 컬럼을 선택하여 가상 뷰를 만드세요.';
                    
                    currentSchemaData = result.schema; // Store schema (expected to be an array of DataSourceSchema objects)
                    displaySelectableSchema(currentSchemaData);
                    virtualViewCreationArea.style.display = 'block'; // Show virtual view form
                } else {
                    connectionMessage.textContent = `오류: ${result.message || response.statusText}`;
                    connectionMessage.style.color = 'red';
                    schemaOutputLoadingStatus.textContent = '스키마 정보를 가져오지 못했습니다.';
                }
            } catch (error) {
                connectionMessage.textContent = `예상치 못한 오류 발생: ${error.message}`;
                connectionMessage.style.color = 'red';
                schemaOutputLoadingStatus.textContent = '스키마 정보를 가져오는 중 오류 발생.';
            }
        });
    }

    function displaySelectableSchema(schema) {
        selectableSchemaContainer.innerHTML = ''; // Clear previous content
        if (!schema || !Array.isArray(schema) || schema.length === 0) {
            selectableSchemaContainer.innerHTML = '<p>표시할 스키마 정보가 없거나 잘못된 형식입니다.</p>';
            return;
        }

        // Group by table_name
        const tables = schema.reduce((acc, col) => {
            const tableName = col.table_name;
            if (!acc[tableName]) {
                acc[tableName] = [];
            }
            acc[tableName].push(col);
            return acc;
        }, {});

        for (const tableName in tables) {
            const tableDiv = document.createElement('div');
            tableDiv.style.marginBottom = '10px';
            
            const tableHeader = document.createElement('h4');
            tableHeader.textContent = tableName;
            tableDiv.appendChild(tableHeader);

            const ul = document.createElement('ul');
            ul.style.listStyleType = 'none';
            ul.style.paddingLeft = '20px';

            tables[tableName].forEach(column => {
                const li = document.createElement('li');
                const checkbox = document.createElement('input');
                checkbox.type = 'checkbox';
                checkbox.id = `col-${column.id}`; // Use DataSourceSchema.id
                checkbox.value = column.id; // Store DataSourceSchema.id in value
                checkbox.name = 'selectedColumns';
                
                const label = document.createElement('label');
                label.htmlFor = `col-${column.id}`;
                label.textContent = ` ${column.column_name} (${column.column_type})`;
                
                li.appendChild(checkbox);
                li.appendChild(label);
                ul.appendChild(li);
            });
            tableDiv.appendChild(ul);
            selectableSchemaContainer.appendChild(tableDiv);
        }
    }

    if (virtualViewForm) {
        virtualViewForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            virtualViewMessage.textContent = '';

            const viewName = virtualViewForm.virtualViewName.value;
            const description = virtualViewForm.virtualViewDescription.value;
            const selectedColumnsCheckboxes = document.querySelectorAll('input[name="selectedColumns"]:checked');
            
            const selectedDataSourceSchemaIDs = [];
            selectedColumnsCheckboxes.forEach(checkbox => {
                selectedDataSourceSchemaIDs.push(checkbox.value); // value is DataSourceSchema.id
            });

            if (!viewName) {
                virtualViewMessage.textContent = '가상 뷰 이름을 입력해주세요.';
                virtualViewMessage.style.color = 'red';
                return;
            }
            if (selectedDataSourceSchemaIDs.length === 0) {
                virtualViewMessage.textContent = '하나 이상의 컬럼을 선택해주세요.';
                virtualViewMessage.style.color = 'red';
                return;
            }

            const token = localStorage.getItem(TOKEN_KEY);
            if (!token) {
                virtualViewMessage.textContent = '오류: 인증 토큰을 찾을 수 없습니다. 다시 로그인해주세요.';
                virtualViewMessage.style.color = 'red';
                return;
            }

            try {
                const response = await fetch('/api/virtual-views/create', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': `Bearer ${token}`
                    },
                    body: JSON.stringify({
                        name: viewName,
                        description: description,
                        selected_schema_ids: selectedDataSourceSchemaIDs 
                    }),
                });

                const result = await response.json();

                if (response.ok) {
                    virtualViewMessage.textContent = `가상 뷰 '${result.name}' (ID: ${result.id})가 성공적으로 생성되었습니다.`;
                    virtualViewMessage.style.color = 'green';
                    // Optionally clear the form or redirect
                    virtualViewForm.reset();
                    selectableSchemaContainer.innerHTML = ''; // Clear schema selection
                    virtualViewCreationArea.style.display = 'none';
                    schemaOutputLoadingStatus.textContent = '새로운 연결을 시도하거나 다른 작업을 수행하세요.';
                    currentSchemaData = null;
                } else {
                    virtualViewMessage.textContent = `오류: ${result.message || response.statusText}`;
                    virtualViewMessage.style.color = 'red';
                }
            } catch (error) {
                virtualViewMessage.textContent = `예상치 못한 오류 발생: ${error.message}`;
                virtualViewMessage.style.color = 'red';
            }
        });
    }

});

// You can add JavaScript to interact with your backend API here.
// Example:
// async function fetchData() {
//     try {
//         const response = await fetch('/api/data?param=someValue');
//         if (!response.ok) {
//             throw new Error(`HTTP error! status: ${response.status}`);
//         }
//         const data = await response.json();
//         console.log(data);
//         // Update your UI with the fetched data
//     } catch (error) {
//         console.error("Could not fetch data:", error);
//     }
// }
// 
// fetchData();
