package core

import (
	"database/sql"

	"Bridgo/internal/models"
)

// CoreService manages core data virtualization logic.
// It acts as a facade that coordinates between different service components.
type CoreService struct {
	metaDB                 *sql.DB // For storing metadata about connected sources, schemas, etc.
	connectionService      *ConnectionService
	virtualViewService     *VirtualViewService
	virtualBaseViewService *VirtualBaseViewService
	dataSourceService      *DataSourceService
	queryService           *QueryService
}

// NewCoreService creates a new core Service.
func NewCoreService(metaDB *sql.DB) *CoreService {
	connectionService := NewConnectionService(metaDB)
	virtualViewService := NewVirtualViewService(metaDB)
	virtualBaseViewService := NewVirtualBaseViewService(metaDB)
	dataSourceService := NewDataSourceService(metaDB)
	queryService := NewQueryService(connectionService)

	return &CoreService{
		metaDB:                 metaDB,
		connectionService:      connectionService,
		virtualViewService:     virtualViewService,
		virtualBaseViewService: virtualBaseViewService,
		dataSourceService:      dataSourceService,
		queryService:           queryService,
	}
}

// Connection related methods
func (s *CoreService) ConnectAndFetchSchema(input ConnectAndFetchSchemaInput) ([]models.DataSourceSchema, error) {
	return s.connectionService.ConnectAndFetchSchema(input)
}

func (s *CoreService) TestConnectionAndFetchSchema(input ConnectAndFetchSchemaInput) ([]models.DataSourceSchema, error) {
	return s.connectionService.TestConnectionAndFetchSchema(input)
}

func (s *CoreService) SaveDataSource(input ConnectAndFetchSchemaInput, schema []models.DataSourceSchema) (*models.DataSource, error) {
	return s.connectionService.SaveDataSource(input, schema)
}

// Virtual View related methods
func (s *CoreService) CreateVirtualView(input CreateVirtualViewInput) (*models.VirtualView, error) {
	return s.virtualViewService.CreateVirtualView(input)
}

func (s *CoreService) GetUserVirtualViews(user_id string) ([]models.VirtualView, error) {
	return s.virtualViewService.GetUserVirtualViews(user_id)
}

func (s *CoreService) GetVirtualViewSchema(virtual_view_id string, user_id string) ([]models.DataSourceSchema, error) {
	return s.virtualViewService.GetVirtualViewSchema(virtual_view_id, user_id)
}

func (s *CoreService) GetVirtualViewSampleData(virtual_view_id string, user_id string) (map[string]interface{}, error) {
	return s.virtualViewService.GetVirtualViewSampleData(virtual_view_id, user_id)
}

// Virtual BaseView related methods
func (s *CoreService) CreateVirtualBaseView(input models.CreateVirtualBaseViewInput) (*models.VirtualBaseView, error) {
	return s.virtualBaseViewService.CreateVirtualBaseView(input)
}

func (s *CoreService) GetUserVirtualBaseViews(userID string) ([]models.VirtualBaseView, error) {
	return s.virtualBaseViewService.GetUserVirtualBaseViews(userID)
}

func (s *CoreService) GetVirtualBaseViewSchema(virtualBaseViewID string, userID string) ([]models.DataSourceSchema, error) {
	return s.virtualBaseViewService.GetVirtualBaseViewSchema(virtualBaseViewID, userID)
}

func (s *CoreService) GetVirtualBaseViewSampleData(virtualBaseViewID string, userID string) (map[string]interface{}, error) {
	return s.virtualBaseViewService.GetVirtualBaseViewSampleData(virtualBaseViewID, userID)
}

// Data Source related methods
func (s *CoreService) GetUserDataSources(user_id string) ([]models.DataSource, error) {
	return s.dataSourceService.GetUserDataSources(user_id)
}

func (s *CoreService) GetDataSourceSchema(data_source_id string, user_id string) ([]models.DataSourceSchema, error) {
	return s.dataSourceService.GetDataSourceSchema(data_source_id, user_id)
}

// Query related methods
func (s *CoreService) QueryData(user_id string, data_source_id string, query string) (interface{}, error) {
	return s.queryService.QueryData(user_id, data_source_id, query)
}
