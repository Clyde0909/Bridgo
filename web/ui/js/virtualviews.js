// virtualviews.js - Virtual BaseView management functionality

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
        this.selectedDataSourceId = null;
        this.selectedTableName = null;
        this.selectedColumns = [];
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
            this.loadUserVirtualBaseViews();
        }
    }

    setupEventListeners() {
        if (this.virtualViewForm) {
            this.virtualViewForm.addEventListener('submit', (e) => this.handleCreateVirtualBaseView(e));
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
                    'Click on a data source to create a Virtual BaseView:' : 
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
        this.selectedDataSourceId = dataSource.id;
        this.selectedTableName = null;
        this.selectedColumns = [];
        
        // Update UI to show selected data source
        this.selectedDataSourceInfo.innerHTML = `
            <h4>Selected Data Source: ${dataSource.source_name}</h4>
            <p><strong>Type:</strong> ${dataSource.db_type} | <strong>Database:</strong> ${dataSource.database_name.String || 'N/A'}</p>
            <p style="color: #666; font-style: italic;">Select a table below to create a Virtual BaseView</p>
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

        // Create instructions
        const instructionsDiv = document.createElement('div');
        instructionsDiv.innerHTML = `
            <h4>Select a Table for Virtual BaseView</h4>
            <p style="color: #666;">Each Virtual BaseView is based on a single table. Click on a table below to select it and choose columns:</p>
        `;
        this.schemaSelectionArea.appendChild(instructionsDiv);

        // Create table selection cards
        for (const tableName in tables) {
            const tableCard = createStyledDiv({ 
                marginBottom: '15px',
                border: '2px solid #ddd',
                borderRadius: '8px',
                padding: '15px',
                cursor: 'pointer',
                backgroundColor: '#f9f9f9',
                transition: 'all 0.3s ease'
            });

            tableCard.innerHTML = `
                <h5 style="margin: 0 0 10px 0; color: #333;">📋 ${tableName}</h5>
                <p style="margin: 0; color: #666;">${tables[tableName].length} columns available</p>
                <p style="margin: 5px 0 0 0; font-size: 12px; color: #888;">Click to select this table and choose columns</p>
            `;

            tableCard.addEventListener('click', () => this.selectTableForBaseView(tableName, tables[tableName]));
            
            // Hover effects
            tableCard.addEventListener('mouseenter', () => {
                if (this.selectedTableName !== tableName) {
                    tableCard.style.backgroundColor = '#e8f4fd';
                    tableCard.style.borderColor = '#007bff';
                }
            });
            tableCard.addEventListener('mouseleave', () => {
                if (this.selectedTableName !== tableName) {
                    tableCard.style.backgroundColor = '#f9f9f9';
                    tableCard.style.borderColor = '#ddd';
                }
            });

            this.schemaSelectionArea.appendChild(tableCard);
        }
    }

    selectTableForBaseView(tableName, columns) {
        this.selectedTableName = tableName;
        this.selectedColumns = [];

        // Update visual feedback for selected table
        const tableCards = this.schemaSelectionArea.querySelectorAll('div[style*="border: 2px solid"]');
        tableCards.forEach(card => {
            if (card.textContent.includes(tableName)) {
                card.style.backgroundColor = '#d4edda';
                card.style.borderColor = '#28a745';
            } else {
                card.style.backgroundColor = '#f9f9f9';
                card.style.borderColor = '#ddd';
            }
        });

        // Display column selection for this table
        this.displayColumnSelection(tableName, columns);
    }

    displayColumnSelection(tableName, columns) {
        // Create or update column selection area
        let columnSelectionArea = document.getElementById('columnSelectionArea');
        if (!columnSelectionArea) {
            columnSelectionArea = document.createElement('div');
            columnSelectionArea.id = 'columnSelectionArea';
            this.schemaSelectionArea.appendChild(columnSelectionArea);
        }

        columnSelectionArea.innerHTML = `
            <div style="margin-top: 20px; padding: 15px; border: 1px solid #ddd; border-radius: 8px; background-color: #f8f9fa;">
                <h5>Select Columns for "${tableName}"</h5>
                <p style="color: #666; margin-bottom: 15px;">Choose which columns to include in your Virtual BaseView:</p>
                <div id="columnCheckboxArea"></div>
                <div style="margin-top: 15px;">
                    <button type="button" id="selectAllColumns" class="btn btn-sm btn-outline-primary">Select All</button>
                    <button type="button" id="clearAllColumns" class="btn btn-sm btn-outline-secondary" style="margin-left: 5px;">Clear All</button>
                </div>
            </div>
        `;

        const columnCheckboxArea = document.getElementById('columnCheckboxArea');
        
        // Create column checkboxes
        columns.forEach(column => {
            const columnDiv = document.createElement('div');
            columnDiv.style.cssText = 'margin-bottom: 8px; padding: 8px; border: 1px solid #e9ecef; border-radius: 4px; background-color: white;';
            
            const checkbox = document.createElement('input');
            checkbox.type = 'checkbox';
            checkbox.id = `col_${column.column_name}`;
            checkbox.value = column.column_name;
            checkbox.addEventListener('change', () => this.updateSelectedColumns());
            
            const isPrimaryKey = column.is_primary_key;
            const allowsNull = column.is_nullable;
            
            columnDiv.innerHTML = `
                <label for="${checkbox.id}" style="display: flex; align-items: center; margin: 0; cursor: pointer;">
                    <input type="checkbox" id="${checkbox.id}" value="${column.column_name}" style="margin-right: 10px;">
                    <div style="flex: 1;">
                        <strong>${column.column_name}</strong>
                        <span style="color: #666; margin-left: 10px;">${column.column_type}</span>
                        ${isPrimaryKey ? '<span style="background: #28a745; color: white; font-size: 10px; padding: 2px 4px; border-radius: 2px; margin-left: 5px;">PK</span>' : ''}
                        ${!allowsNull ? '<span style="background: #dc3545; color: white; font-size: 10px; padding: 2px 4px; border-radius: 2px; margin-left: 5px;">NOT NULL</span>' : ''}
                    </div>
                </label>
            `;
            
            columnCheckboxArea.appendChild(columnDiv);
        });

        // Setup select all / clear all buttons
        document.getElementById('selectAllColumns').addEventListener('click', () => {
            const checkboxes = columnCheckboxArea.querySelectorAll('input[type="checkbox"]');
            checkboxes.forEach(cb => cb.checked = true);
            this.updateSelectedColumns();
        });

        document.getElementById('clearAllColumns').addEventListener('click', () => {
            const checkboxes = columnCheckboxArea.querySelectorAll('input[type="checkbox"]');
            checkboxes.forEach(cb => cb.checked = false);
            this.updateSelectedColumns();
        });
    }

    updateSelectedColumns() {
        const checkboxes = document.querySelectorAll('#columnCheckboxArea input[type="checkbox"]:checked');
        this.selectedColumns = Array.from(checkboxes).map(cb => cb.value);
        console.log('Selected columns:', this.selectedColumns);
    }

    async handleCreateVirtualBaseView(e) {
        e.preventDefault();
        displayMessage(this.virtualViewMessage, '');

        const viewName = this.virtualViewForm.virtualViewName.value;
        const description = this.virtualViewForm.virtualViewDescription.value;

        if (!viewName) {
            displayMessage(this.virtualViewMessage, 'Error: Please enter a view name.', 'error');
            return;
        }
        
        if (!this.selectedDataSourceId) {
            displayMessage(this.virtualViewMessage, 'Error: Please select a data source.', 'error');
            return;
        }

        if (!this.selectedTableName) {
            displayMessage(this.virtualViewMessage, 'Error: Please select a table.', 'error');
            return;
        }

        if (this.selectedColumns.length === 0) {
            displayMessage(this.virtualViewMessage, 'Error: Please select at least one column.', 'error');
            return;
        }

        const token = getAuthToken();
        if (!token) {
            displayMessage(this.virtualViewMessage, 'Error: Please log in first.', 'error');
            return;
        }

        try {
            const response = await fetch('/api/virtual-base-views', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${token}`
                },
                body: JSON.stringify({
                    name: viewName,
                    description: description,
                    data_source_id: this.selectedDataSourceId,
                    table_name: this.selectedTableName,
                    selected_columns: this.selectedColumns
                }),
            });

            const result = await response.json();

            if (response.ok) {
                displayMessage(this.virtualViewMessage, 'Virtual BaseView created successfully!', 'success');
                this.virtualViewForm.reset();
                
                // Clear selection state
                this.selectedDataSourceId = null;
                this.selectedTableName = null;
                this.selectedColumns = [];
                
                // Clear UI
                clearElement(this.schemaSelectionArea);
                if (this.virtualViewCreationArea) {
                    this.virtualViewCreationArea.style.display = 'none';
                }
                clearElement(this.selectedDataSourceInfo);
                
                // Reload virtual base views list
                this.loadUserVirtualBaseViews();
            } else {
                displayMessage(this.virtualViewMessage, `Error: ${result.message || response.statusText}`, 'error');
            }
        } catch (error) {
            displayMessage(this.virtualViewMessage, `Unexpected error occurred: ${error.message}`, 'error');
        }
    }

    async loadUserVirtualBaseViews() {
        try {
            const token = getAuthToken();
            if (!token) {
                displayMessage(this.virtualViewsLoadingStatus, 'Error: Please log in first.', 'error');
                return;
            }

            const response = await fetch('/api/virtual-base-views', {
                method: 'GET',
                headers: {
                    'Authorization': `Bearer ${token}`
                }
            });

            const result = await response.json();

            if (response.ok) {
                this.displayVirtualBaseViews(result.virtual_base_views || []);
                const message = (result.virtual_base_views && result.virtual_base_views.length > 0) ? 
                    'Your Virtual BaseViews:' : 
                    'No Virtual BaseViews found.';
                displayMessage(this.virtualViewsLoadingStatus, message, 'info');
            } else {
                displayMessage(this.virtualViewsLoadingStatus, `Error: ${result.message || response.statusText}`, 'error');
            }
        } catch (error) {
            displayMessage(this.virtualViewsLoadingStatus, `Error loading Virtual BaseViews: ${error.message}`, 'error');
        }
    }

    displayVirtualBaseViews(virtualBaseViews) {
        clearElement(this.virtualViewsList);
        
        if (!virtualBaseViews || virtualBaseViews.length === 0) {
            this.virtualViewsList.innerHTML = '<p>No Virtual BaseViews available.</p>';
            return;
        }

        virtualBaseViews.forEach(vbv => {
            const vbvDiv = createStyledDiv({
                border: '1px solid #ccc',
                margin: '10px 0',
                padding: '10px',
                borderRadius: '5px',
                backgroundColor: '#f0f8ff',
                cursor: 'pointer'
            });

            const selectedColumns = vbv.selected_columns ? JSON.parse(vbv.selected_columns) : [];
            
            vbvDiv.innerHTML = `
                <h4>${vbv.name}</h4>
                <p><strong>Description:</strong> ${vbv.description || 'No description'}</p>
                <p><strong>Table:</strong> ${vbv.table_name}</p>
                <p><strong>Columns:</strong> ${selectedColumns.length} columns selected</p>
                <p><strong>Created:</strong> ${new Date(vbv.created_at).toLocaleString()}</p>
                <p style="margin-top: 10px; font-style: italic; color: #666;">Click to view details</p>
            `;

            vbvDiv.addEventListener('click', () => this.showVirtualBaseViewDetails(vbv));
            this.virtualViewsList.appendChild(vbvDiv);
        });
    }

    async showVirtualBaseViewDetails(virtualBaseView) {
        try {
            const token = getAuthToken();
            if (!token) {
                alert('Error: Please log in first.');
                return;
            }

            // Load schema
            const schemaResponse = await fetch(`/api/virtual-base-views/schema?virtual_base_view_id=${virtualBaseView.id}`, {
                method: 'GET',
                headers: {
                    'Authorization': `Bearer ${token}`
                }
            });

            if (!schemaResponse.ok) {
                const error = await schemaResponse.json();
                alert(`Error loading schema: ${error.message || schemaResponse.statusText}`);
                return;
            }

            const schemaResult = await schemaResponse.json();
            this.displayVirtualBaseViewModal(virtualBaseView, schemaResult.schema);

        } catch (error) {
            alert(`Error loading Virtual BaseView details: ${error.message}`);
        }
    }

    displayVirtualBaseViewModal(virtualBaseView, schema) {
        // Create modal
        const modal = document.createElement('div');
        modal.style.cssText = `
            position: fixed; top: 0; left: 0; width: 100%; height: 100%; 
            background-color: rgba(0,0,0,0.5); display: flex; 
            justify-content: center; align-items: center; z-index: 1000;
        `;

        const modalContent = document.createElement('div');
        modalContent.style.cssText = `
            background: white; padding: 20px; border-radius: 8px; 
            max-width: 90%; max-height: 90%; overflow-y: auto;
            min-width: 600px;
        `;

        const selectedColumns = virtualBaseView.selected_columns ? JSON.parse(virtualBaseView.selected_columns) : [];

        modalContent.innerHTML = `
            <div style="display: flex; justify-content: between; align-items: center; margin-bottom: 20px;">
                <h3 style="margin: 0;">${virtualBaseView.name}</h3>
                <button id="closeModal" style="background: #dc3545; color: white; border: none; padding: 8px 12px; border-radius: 4px; cursor: pointer; margin-left: 20px;">Close</button>
            </div>
            
            <div style="margin-bottom: 20px;">
                <p><strong>Description:</strong> ${virtualBaseView.description || 'No description'}</p>
                <p><strong>Table:</strong> ${virtualBaseView.table_name}</p>
                <p><strong>Columns:</strong> ${selectedColumns.length} selected</p>
                <p><strong>Created:</strong> ${new Date(virtualBaseView.created_at).toLocaleString()}</p>
            </div>

            <div style="margin-bottom: 20px;">
                <button id="viewSchemaBtn" class="btn btn-primary" style="margin-right: 10px;">View Schema</button>
                <button id="viewSampleDataBtn" class="btn btn-secondary">View Sample Data</button>
            </div>

            <div id="modalContentArea"></div>
        `;

        modal.appendChild(modalContent);
        document.body.appendChild(modal);

        // Event listeners
        document.getElementById('closeModal').addEventListener('click', () => {
            document.body.removeChild(modal);
        });

        modal.addEventListener('click', (e) => {
            if (e.target === modal) {
                document.body.removeChild(modal);
            }
        });

        document.getElementById('viewSchemaBtn').addEventListener('click', () => {
            this.displaySchemaInModal(schema);
        });

        document.getElementById('viewSampleDataBtn').addEventListener('click', () => {
            this.loadSampleDataInModal(virtualBaseView.id);
        });

        // Auto-load schema
        this.displaySchemaInModal(schema);
    }

    displaySchemaInModal(schema) {
        const contentArea = document.getElementById('modalContentArea');
        
        if (!schema || schema.length === 0) {
            contentArea.innerHTML = '<p>No schema information available.</p>';
            return;
        }

        let tableHTML = `
            <h4>Schema Details</h4>
            <table style="width: 100%; border-collapse: collapse; margin-top: 10px;">
                <thead>
                    <tr style="background-color: #f8f9fa;">
                        <th style="border: 1px solid #ddd; padding: 8px; text-align: left;">Column Name</th>
                        <th style="border: 1px solid #ddd; padding: 8px; text-align: left;">Type</th>
                        <th style="border: 1px solid #ddd; padding: 8px; text-align: center;">Primary Key</th>
                        <th style="border: 1px solid #ddd; padding: 8px; text-align: center;">Allow Null</th>
                    </tr>
                </thead>
                <tbody>
        `;

        schema.forEach(column => {
            const isPrimaryKey = column.is_primary_key;
            const allowsNull = column.is_nullable;
            
            tableHTML += `
                <tr>
                    <td style="border: 1px solid #ddd; padding: 8px;">${column.column_name}</td>
                    <td style="border: 1px solid #ddd; padding: 8px;">${column.column_type}</td>
                    <td style="border: 1px solid #ddd; padding: 8px; text-align: center;">
                        <span style="color: ${isPrimaryKey ? 'green' : 'red'};">${isPrimaryKey ? '✓' : '✗'}</span>
                    </td>
                    <td style="border: 1px solid #ddd; padding: 8px; text-align: center;">
                        <span style="color: ${allowsNull ? 'green' : 'red'};">${allowsNull ? '✓' : '✗'}</span>
                    </td>
                </tr>
            `;
        });

        tableHTML += '</tbody></table>';
        contentArea.innerHTML = tableHTML;
    }

    async loadSampleDataInModal(virtualBaseViewId) {
        const contentArea = document.getElementById('modalContentArea');
        contentArea.innerHTML = '<p>Loading sample data...</p>';

        try {
            const token = getAuthToken();
            const response = await fetch(`/api/virtual-base-views/sample-data?virtual_base_view_id=${virtualBaseViewId}`, {
                method: 'GET',
                headers: {
                    'Authorization': `Bearer ${token}`
                }
            });

            const result = await response.json();

            if (response.ok) {
                this.displaySampleDataInModal(result);
            } else {
                contentArea.innerHTML = `<p>Error loading sample data: ${result.message || response.statusText}</p>`;
            }
        } catch (error) {
            contentArea.innerHTML = `<p>Error loading sample data: ${error.message}</p>`;
        }
    }

    displaySampleDataInModal(data) {
        const contentArea = document.getElementById('modalContentArea');
        
        if (!data || !data.columns || data.columns.length === 0) {
            contentArea.innerHTML = '<p>No sample data available.</p>';
            return;
        }

        let tableHTML = `
            <h4>Sample Data (First 5 rows)</h4>
            <div style="overflow-x: auto;">
                <table style="width: 100%; border-collapse: collapse; margin-top: 10px;">
                    <thead>
                        <tr style="background-color: #f8f9fa;">
        `;

        // Add column headers
        data.columns.forEach(columnName => {
            tableHTML += `<th style="border: 1px solid #ddd; padding: 8px; text-align: left; white-space: nowrap;">${columnName}</th>`;
        });

        tableHTML += '</tr></thead><tbody>';

        // Add data rows
        if (data.rows && data.rows.length > 0) {
            data.rows.forEach(row => {
                tableHTML += '<tr>';
                data.columns.forEach(columnName => {
                    const cellValue = row[columnName];
                    const displayValue = cellValue !== null && cellValue !== undefined ? String(cellValue) : '<em>NULL</em>';
                    tableHTML += `<td style="border: 1px solid #ddd; padding: 8px; white-space: nowrap;">${displayValue}</td>`;
                });
                tableHTML += '</tr>';
            });
        } else {
            tableHTML += `<tr><td colspan="${data.columns.length}" style="border: 1px solid #ddd; padding: 8px; text-align: center; font-style: italic;">No data available</td></tr>`;
        }

        tableHTML += '</tbody></table></div>';
        contentArea.innerHTML = tableHTML;
    }
}

// Initialize when DOM is loaded
document.addEventListener('DOMContentLoaded', function() {
    const virtualViewManager = new VirtualViewManager();
    virtualViewManager.init();
});
