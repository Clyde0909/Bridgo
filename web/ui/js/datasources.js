// datasources.js - Data source management functionality

class DataSourceManager {
    constructor() {
        this.db_connection_form = null;
        this.connection_message = null;
        this.schema_output_loading_status = null;
        this.selectable_schema_container = null;
        this.save_data_source_btn = null;
        this.save_data_source_message = null;
        this.current_schema_data = null;
        this.current_connection_data = null;
    }

    init() {
        this.db_connection_form = document.getElementById('dbConnectionForm');
        this.connection_message = document.getElementById('connectionMessage');
        this.schema_output_loading_status = document.getElementById('schemaOutputLoadingStatus');
        this.selectable_schema_container = document.getElementById('selectableSchemaContainer');
        this.save_data_source_btn = document.getElementById('saveDataSourceBtn');
        this.save_data_source_message = document.getElementById('saveDataSourceMessage');

        this.setupEventListeners();
    }

    setupEventListeners() {
        if (this.db_connection_form) {
            this.db_connection_form.addEventListener('submit', (e) => this.handleConnectionTest(e));
        }

        if (this.save_data_source_btn) {
            this.save_data_source_btn.addEventListener('click', () => this.handleSaveDataSource());
        }
    }

    async handleConnectionTest(e) {
        e.preventDefault();
        
        displayMessage(this.connection_message, '');
        clearElement(this.selectable_schema_container);
        
        const saveDataSourceArea = document.getElementById('saveDataSourceArea');
        if (saveDataSourceArea) {
            saveDataSourceArea.style.display = 'none';
        }
        
        displayMessage(this.schema_output_loading_status, 'Testing connection and fetching schema...', 'info');
        this.current_schema_data = null;

        const form_data = new FormData(this.db_connection_form);
        const data = Object.fromEntries(form_data.entries());
        
        if (data.dbPort) {
            data.dbPort = parseInt(data.dbPort, 10);
        }

        try {
            const token = getAuthToken();
            if (!token) {
                displayMessage(this.connection_message, 'Error: Please log in first.', 'error');
                return;
            }

            const response = await fetch('/api/db/test-connection', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${token}`
                },
                body: JSON.stringify(data),
            });

            const result = await response.json();

            if (response.ok) {
                displayMessage(this.connection_message, 'Connection successful!', 'success');
                displayMessage(this.schema_output_loading_status, 'Schema loaded successfully!', 'success');
                
                this.current_connection_data = data;
                this.current_schema_data = result.schema;
                this.displaySelectableSchema(result.schema);
                
                if (saveDataSourceArea) {
                    saveDataSourceArea.style.display = 'block';
                }
            } else {
                displayMessage(this.connection_message, `Connection failed: ${result.message || response.statusText}`, 'error');
                displayMessage(this.schema_output_loading_status, 'Failed to load schema.', 'error');
            }
        } catch (error) {
            displayMessage(this.connection_message, `Unexpected error occurred: ${error.message}`, 'error');
            displayMessage(this.schema_output_loading_status, 'Failed to load schema.', 'error');
        }
    }

    displaySelectableSchema(schema) {
        clearElement(this.selectable_schema_container);
        
        if (!schema || !Array.isArray(schema) || schema.length === 0) {
            this.selectable_schema_container.innerHTML = '<p>No schema information available or invalid format.</p>';
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
            const tableSection = createStyledDiv({ 
                marginBottom: '20px',
                border: '1px solid #ddd',
                borderRadius: '5px',
                overflow: 'hidden'
            });
            
            // Table header
            const tableHeader = document.createElement('h4');
            tableHeader.textContent = `Table: ${tableName}`;
            tableHeader.style.margin = '0';
            tableHeader.style.padding = '10px 15px';
            tableHeader.style.backgroundColor = '#f5f5f5';
            tableHeader.style.borderBottom = '1px solid #ddd';
            tableSection.appendChild(tableHeader);

            // Create table
            const table = document.createElement('table');
            table.style.width = '100%';
            table.style.borderCollapse = 'collapse';
            table.style.fontSize = '14px';

            // Create table header
            const thead = document.createElement('thead');
            const headerRow = document.createElement('tr');
            headerRow.style.backgroundColor = '#f8f9fa';
            
            const headers = ['Schema Name', 'Schema Type', 'PK', 'Allow Null'];
            headers.forEach(headerText => {
                const th = document.createElement('th');
                th.textContent = headerText;
                th.style.padding = '12px 8px';
                th.style.textAlign = 'left';
                th.style.borderBottom = '2px solid #dee2e6';
                th.style.fontWeight = 'bold';
                th.style.color = '#495057';
                headerRow.appendChild(th);
            });
            
            thead.appendChild(headerRow);
            table.appendChild(thead);

            // Create table body
            const tbody = document.createElement('tbody');
            
            tables[tableName].forEach((column, index) => {
                const row = document.createElement('tr');
                row.style.backgroundColor = index % 2 === 0 ? '#fff' : '#f8f9fa';
                row.style.borderBottom = '1px solid #dee2e6';
                
                // Schema Name
                const nameCell = document.createElement('td');
                nameCell.textContent = column.column_name || 'N/A';
                nameCell.style.padding = '10px 8px';
                nameCell.style.fontWeight = '500';
                row.appendChild(nameCell);
                
                // Schema Type
                const typeCell = document.createElement('td');
                typeCell.textContent = column.column_type || 'N/A';
                typeCell.style.padding = '10px 8px';
                typeCell.style.color = '#6c757d';
                row.appendChild(typeCell);
                
                // Primary Key
                const pkCell = document.createElement('td');
                const isPrimaryKey = column.is_primary_key || column.primary_key || false;
                console.log('isPrimaryKey:', column.IsPrimaryKey);
                // Handle different possible values for primary key
                let pkValue = '✗';
                if (isPrimaryKey && isPrimaryKey.Valid === true && isPrimaryKey.Bool === true) {
                    pkValue = '✓';
                }
                pkCell.textContent = pkValue;
                pkCell.style.padding = '10px 8px';
                pkCell.style.textAlign = 'center';
                pkCell.style.color = pkValue === '✓' ? '#28a745' : '#dc3545';
                pkCell.style.fontWeight = 'bold';
                row.appendChild(pkCell);
                
                // Allow Null
                const nullCell = document.createElement('td');
                const allowsNull = column.is_nullable || column.nullable || column.allow_null;
                // Handle different possible values for nullable
                let nullValue = '✗';
                if (allowsNull && allowsNull.Valid === true && allowsNull.Bool === true) {
                    nullValue = '✓';
                }
                nullCell.textContent = nullValue;
                nullCell.style.padding = '10px 8px';
                nullCell.style.textAlign = 'center';
                nullCell.style.color = nullValue === '✓' ? '#28a745' : '#dc3545';
                nullCell.style.fontWeight = 'bold';
                row.appendChild(nullCell);
                
                tbody.appendChild(row);
            });
            
            table.appendChild(tbody);
            tableSection.appendChild(table);
            this.selectable_schema_container.appendChild(tableSection);
        }
    }

    async handleSaveDataSource() {
        if (!this.current_connection_data || !this.current_schema_data) {
            displayMessage(this.save_data_source_message, 'Error: No connection data or schema available to save.', 'error');
            return;
        }

        displayMessage(this.save_data_source_message, 'Saving data source...', 'info');

        try {
            const token = getAuthToken();
            if (!token) {
                displayMessage(this.save_data_source_message, 'Error: Please log in first.', 'error');
                return;
            }

            const response = await fetch('/api/db/save-datasource', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${token}`
                },
                body: JSON.stringify({
                    connection: this.current_connection_data,
                    schema: this.current_schema_data
                }),
            });

            const result = await response.json();

            if (response.ok) {
                displayMessage(this.save_data_source_message, 'Data source saved successfully!', 'success');
            } else {
                displayMessage(this.save_data_source_message, `Error: ${result.message || response.statusText}`, 'error');
            }
        } catch (error) {
            displayMessage(this.save_data_source_message, `Unexpected error occurred: ${error.message}`, 'error');
        }
    }
}

if (typeof module !== 'undefined' && module.exports) {
    module.exports = DataSourceManager;
}
