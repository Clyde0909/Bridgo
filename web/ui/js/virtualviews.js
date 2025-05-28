// virtualviews.js - Virtual view management functionality

class VirtualViewManager {
    constructor() {
        this.virtualViewForm = null;
        this.virtualViewMessage = null;
        this.virtualViewCreationArea = null;
        this.dataSourcesList = null;
        this.dataSourcesLoadingStatus = null;
        this.virtualViewsLoadingStatus = null;
        this.virtualViewsList = null;
        this.schemaSelectionArea = null;
        this.selectedDataSourceInfo = null;
    }

    init() {
        this.virtualViewForm = document.getElementById('virtualViewForm');
        this.virtualViewMessage = document.getElementById('virtualViewMessage');
        this.virtualViewCreationArea = document.getElementById('virtualViewCreationArea');
        this.dataSourcesList = document.getElementById('dataSourcesList');
        this.dataSourcesLoadingStatus = document.getElementById('dataSourcesLoadingStatus');
        this.virtualViewsLoadingStatus = document.getElementById('virtualViewsLoadingStatus');
        this.virtualViewsList = document.getElementById('virtualViewsList');
        this.schemaSelectionArea = document.getElementById('schemaSelectionArea');
        this.selectedDataSourceInfo = document.getElementById('selectedDataSourceInfo');

        this.setupEventListeners();

        if (window.location.pathname.includes('/virtual_views')) {
            this.loadUserDataSources();
            this.loadUserVirtualViews();
        }
    }

    setupEventListeners() {
        if (this.virtualViewForm) {
            this.virtualViewForm.addEventListener('submit', (e) => this.handleCreateVirtualView(e));
        }
    }

    async loadUserDataSources() {
        try {
            const token = getAuthToken();
            if (!token) {
                displayMessage(this.dataSourcesLoadingStatus, 'Error: Please log in first.', 'error');
                return;
            }

            const response = await fetch('/api/datasources', {
                method: 'GET',
                headers: {
                    'Authorization': `Bearer ${token}`
                }
            });

            const result = await response.json();

            if (response.ok) {
                this.displayDataSources(result.datasources);
                const message = result.datasources.length > 0 ? 
                    'Click on a data source to create a virtual view:' : 
                    'No saved data sources found. Please add some data sources first.';
                displayMessage(this.dataSourcesLoadingStatus, message, 'info');
            } else {
                displayMessage(this.dataSourcesLoadingStatus, `Error: ${result.message || response.statusText}`, 'error');
            }
        } catch (error) {
            displayMessage(this.dataSourcesLoadingStatus, `Error loading data sources: ${error.message}`, 'error');
        }
    }

    displayDataSources(dataSources) {
        clearElement(this.dataSourcesList);
        
        if (!dataSources || dataSources.length === 0) {
            this.dataSourcesList.innerHTML = '<p>No data sources available.</p>';
            return;
        }

        dataSources.forEach(ds => {
            const dsDiv = createStyledDiv({
                border: '1px solid #ccc',
                margin: '10px 0',
                padding: '10px',
                borderRadius: '5px',
                cursor: 'pointer',
                backgroundColor: '#f9f9f9'
            });

            dsDiv.innerHTML = `
                <h4>${ds.source_name}</h4>
                <p><strong>Type:</strong> ${ds.db_type}</p>
                <p><strong>Host:</strong> ${ds.host.String || 'N/A'}</p>
                <p><strong>Database:</strong> ${ds.database_name.String || 'N/A'}</p>
                <p><strong>Created:</strong> ${new Date(ds.created_at).toLocaleString()}</p>
            `;

            dsDiv.addEventListener('click', () => this.selectDataSource(ds));
            this.dataSourcesList.appendChild(dsDiv);
        });
    }

    async selectDataSource(dataSource) {
        // Update UI to show selected data source
        this.selectedDataSourceInfo.innerHTML = `
            <h4>Selected Data Source: ${dataSource.source_name}</h4>
            <p><strong>Type:</strong> ${dataSource.db_type} | <strong>Database:</strong> ${dataSource.database_name.String || 'N/A'}</p>
        `;

        // Load schema for this data source
        await this.loadDataSourceSchema(dataSource.id);
        
        // Show virtual view creation area
        if (this.virtualViewCreationArea) {
            this.virtualViewCreationArea.style.display = 'block';
        }
    }

    async loadDataSourceSchema(dataSourceId) {
        try {
            const token = getAuthToken();
            if (!token) {
                this.schemaSelectionArea.innerHTML = '<p>Error: Please log in first.</p>';
                return;
            }

            this.schemaSelectionArea.innerHTML = '<p>Loading schema...</p>';

            const response = await fetch(`/api/datasources/schema?datasource_id=${dataSourceId}`, {
                method: 'GET',
                headers: {
                    'Authorization': `Bearer ${token}`
                }
            });

            const result = await response.json();

            if (response.ok) {
                this.displaySchemaForSelection(result.schema, dataSourceId);
            } else {
                this.schemaSelectionArea.innerHTML = `<p>Error loading schema: ${result.message || response.statusText}</p>`;
            }
        } catch (error) {
            this.schemaSelectionArea.innerHTML = `<p>Error loading schema: ${error.message}</p>`;
        }
    }

    displaySchemaForSelection(schema, dataSourceId) {
        clearElement(this.schemaSelectionArea);
        
        if (!schema || schema.length === 0) {
            this.schemaSelectionArea.innerHTML = '<p>No schema information available.</p>';
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
            
            // Table header with checkbox for selecting entire table
            const tableHeaderDiv = document.createElement('div');
            tableHeaderDiv.className = 'table-header-with-checkbox';
            
            const tableCheckbox = document.createElement('input');
            tableCheckbox.type = 'checkbox';
            tableCheckbox.id = `table_${tableName.replace(/[^a-zA-Z0-9]/g, '_')}`;
            tableCheckbox.addEventListener('change', () => {
                const columnCheckboxes = tableSection.querySelectorAll('.column-checkbox');
                columnCheckboxes.forEach(cb => cb.checked = tableCheckbox.checked);
            });
            
            const tableLabel = document.createElement('label');
            tableLabel.htmlFor = tableCheckbox.id;
            tableLabel.textContent = `Table: ${tableName}`;
            
            tableHeaderDiv.appendChild(tableCheckbox);
            tableHeaderDiv.appendChild(tableLabel);
            tableSection.appendChild(tableHeaderDiv);

            // Create table
            const table = document.createElement('table');
            table.className = 'schema-table';

            // Create table header
            const thead = document.createElement('thead');
            const headerRow = document.createElement('tr');
            
            const headers = ['Select', 'Column Name', 'Column Type', 'PK', 'Allow Null'];
            headers.forEach(headerText => {
                const th = document.createElement('th');
                th.textContent = headerText;
                headerRow.appendChild(th);
            });
            
            thead.appendChild(headerRow);
            table.appendChild(thead);

            // Create table body
            const tbody = document.createElement('tbody');
            
            tables[tableName].forEach((column, index) => {
                const row = document.createElement('tr');
                
                // Select checkbox
                const selectCell = document.createElement('td');
                selectCell.className = 'select-cell';
                
                const checkbox = document.createElement('input');
                checkbox.type = 'checkbox';
                checkbox.id = `schema_${column.id}`;
                checkbox.value = column.id;
                checkbox.name = 'selectedColumns';
                checkbox.className = 'column-checkbox';
                checkbox.addEventListener('change', () => {
                    // Update table checkbox state based on column selections
                    const allColumnCheckboxes = tableSection.querySelectorAll('.column-checkbox');
                    const checkedColumnCheckboxes = tableSection.querySelectorAll('.column-checkbox:checked');
                    
                    if (checkedColumnCheckboxes.length === 0) {
                        tableCheckbox.checked = false;
                        tableCheckbox.indeterminate = false;
                    } else if (checkedColumnCheckboxes.length === allColumnCheckboxes.length) {
                        tableCheckbox.checked = true;
                        tableCheckbox.indeterminate = false;
                    } else {
                        tableCheckbox.checked = false;
                        tableCheckbox.indeterminate = true;
                    }
                });
                
                selectCell.appendChild(checkbox);
                row.appendChild(selectCell);
                
                // Column Name
                const nameCell = document.createElement('td');
                nameCell.className = 'column-name';
                nameCell.textContent = column.column_name || 'N/A';
                row.appendChild(nameCell);
                
                // Column Type
                const typeCell = document.createElement('td');
                typeCell.className = 'column-type';
                typeCell.textContent = column.column_type || 'N/A';
                row.appendChild(typeCell);
                
                // Primary Key
                const pkCell = document.createElement('td');
                const isPrimaryKey = column.is_primary_key || column.primary_key || false;
                pkCell.className = `pk-cell ${isPrimaryKey ? 'pk-yes' : 'pk-no'}`;
                pkCell.textContent = isPrimaryKey ? '✓' : '✗';
                row.appendChild(pkCell);
                
                // Allow Null
                const nullCell = document.createElement('td');
                const allowsNull = column.is_nullable || column.nullable || column.allow_null;
                let nullValue = '✗';
                let nullClass = 'null-no';
                if (allowsNull === true || allowsNull === 'YES' || allowsNull === 'Y' || allowsNull === 1) {
                    nullValue = '✓';
                    nullClass = 'null-yes';
                }
                nullCell.className = `null-cell ${nullClass}`;
                nullCell.textContent = nullValue;
                row.appendChild(nullCell);
                
                tbody.appendChild(row);
            });
            
            table.appendChild(tbody);
            tableSection.appendChild(table);
            this.schemaSelectionArea.appendChild(tableSection);
        }
    }

    async handleCreateVirtualView(e) {
        e.preventDefault();
        displayMessage(this.virtualViewMessage, '');

        const viewName = this.virtualViewForm.virtualViewName.value;
        const description = this.virtualViewForm.virtualViewDescription.value;
        const selectedColumnsCheckboxes = document.querySelectorAll('input[name="selectedColumns"]:checked');
        
        const selectedDataSourceSchemaIDs = [];
        selectedColumnsCheckboxes.forEach(checkbox => {
            selectedDataSourceSchemaIDs.push(parseInt(checkbox.value, 10));
        });

        if (!viewName) {
            displayMessage(this.virtualViewMessage, 'Error: Please enter a view name.', 'error');
            return;
        }
        
        if (selectedDataSourceSchemaIDs.length === 0) {
            displayMessage(this.virtualViewMessage, 'Error: Please select at least one column.', 'error');
            return;
        }

        const token = getAuthToken();
        if (!token) {
            displayMessage(this.virtualViewMessage, 'Error: Please log in first.', 'error');
            return;
        }

        try {
            const response = await fetch('/api/virtual-views', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${token}`
                },
                body: JSON.stringify({
                    name: viewName,
                    description: description,
                    selectedDataSourceSchemaIDs: selectedDataSourceSchemaIDs
                }),
            });

            const result = await response.json();

            if (response.ok) {
                displayMessage(this.virtualViewMessage, 'Virtual view created successfully!', 'success');
                this.virtualViewForm.reset();
                // Reload virtual views list
                this.loadUserVirtualViews();
            } else {
                displayMessage(this.virtualViewMessage, `Error: ${result.message || response.statusText}`, 'error');
            }
        } catch (error) {
            displayMessage(this.virtualViewMessage, `Unexpected error occurred: ${error.message}`, 'error');
        }
    }

    async loadUserVirtualViews() {
        try {
            const token = getAuthToken();
            if (!token) {
                displayMessage(this.virtualViewsLoadingStatus, 'Error: Please log in first.', 'error');
                return;
            }

            const response = await fetch('/api/virtual-views', {
                method: 'GET',
                headers: {
                    'Authorization': `Bearer ${token}`
                }
            });

            const result = await response.json();

            if (response.ok) {
                this.displayVirtualViews(result.virtualviews);
                const message = result.virtualviews.length > 0 ? 
                    'Your virtual views:' : 
                    'No virtual views found.';
                displayMessage(this.virtualViewsLoadingStatus, message, 'info');
            } else {
                displayMessage(this.virtualViewsLoadingStatus, `Error: ${result.message || response.statusText}`, 'error');
            }
        } catch (error) {
            displayMessage(this.virtualViewsLoadingStatus, `Error loading virtual views: ${error.message}`, 'error');
        }
    }

    displayVirtualViews(virtualViews) {
        clearElement(this.virtualViewsList);
        
        if (!virtualViews || virtualViews.length === 0) {
            this.virtualViewsList.innerHTML = '<p>No virtual views available.</p>';
            return;
        }

        virtualViews.forEach(vv => {
            const vvDiv = createStyledDiv({
                border: '1px solid #ccc',
                margin: '10px 0',
                padding: '10px',
                borderRadius: '5px',
                backgroundColor: '#f0f8ff'
            });

            const definition = JSON.parse(vv.definition);
            
            vvDiv.innerHTML = `
                <h4>${vv.name}</h4>
                <p><strong>Description:</strong> ${vv.description || 'No description'}</p>
                <p><strong>Tables:</strong> ${definition.selected_tables ? definition.selected_tables.join(', ') : 'N/A'}</p>
                <p><strong>Columns:</strong> ${definition.selected_columns ? definition.selected_columns.length : 0} columns</p>
                <p><strong>Created:</strong> ${new Date(vv.created_at).toLocaleString()}</p>
            `;

            this.virtualViewsList.appendChild(vvDiv);
        });
    }
}

if (typeof module !== 'undefined' && module.exports) {
    module.exports = VirtualViewManager;
}
