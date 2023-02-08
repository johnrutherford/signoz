package pipelines

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"go.signoz.io/signoz/ee/query-service/model"
	"go.signoz.io/signoz/ee/query-service/pipelines/sqlite"
	"go.uber.org/zap"
)

// Repo handles DDL and DML ops on ingestion pipeline
type Repo struct {
	db *sqlx.DB
}

const logPipelines = "log_pipelines"

// NewRepo initiates a new ingestion repo
func NewRepo(db *sqlx.DB) Repo {
	return Repo{
		db: db,
	}
}

func (r *Repo) InitDB(engine string) error {
	switch engine {
	case "sqlite3", "sqlite":
		return sqlite.InitDB(r.db)
	default:
		return fmt.Errorf("unsupported db")
	}
}

// insertPipeline stores a given postable pipeline to database
func (r *Repo) insertPipeline(ctx context.Context, postable *PostablePipeline) (*model.Pipeline, error) {
	if err := postable.IsValid(); err != nil {
		return nil, errors.Wrap(err, "failed to validate postable ingestion pipeline")
	}

	rawConfig, err := json.Marshal(postable.Config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal postable ingestion config")
	}

	insertRow := &model.Pipeline{
		Id:        uuid.New().String(),
		OrderId:   postable.OrderId,
		Enabled:   postable.Enabled,
		Name:      postable.Name,
		Alias:     postable.Alias,
		Filter:    postable.Filter,
		Config:    postable.Config,
		RawConfig: string(rawConfig),
	}

	insertQuery := `INSERT INTO pipelines 
	(id, order_id, enabled, name, alias, filter, config_json) 
	VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err = r.db.ExecContext(ctx,
		insertQuery,
		insertRow.Id,
		insertRow.OrderId,
		insertRow.Enabled,
		insertRow.Name,
		insertRow.Alias,
		insertRow.Filter,
		insertRow.RawConfig)

	if err != nil {
		zap.S().Errorf("error in inserting pipeline data: ", zap.Error(err))
		return insertRow, errors.Wrap(err, "failed to insert pipeline")
	}

	return insertRow, nil
}

// getPipelinesByVersion returns pipelines associated with a given version
func (r *Repo) getPipelinesByVersion(ctx context.Context, version int) ([]model.Pipeline, []error) {
	var errors []error
	pipelines := []model.Pipeline{}

	versionQuery := `SELECT r.id, 
		r.name, 
		r.config_json, 
		r.deployment_status, 
		r.deployment_sequence 
		FROM pipelines r,
			 agent_config_elements e,
			 agent_config_versions v
		WHERE r.id = e.element_id
		AND v.id = e.version_id
		AND e.element_type = $1
		AND v.version = $2`

	err := r.db.SelectContext(ctx, &pipelines, versionQuery, logPipelines, version)
	if err != nil {
		return nil, []error{fmt.Errorf("failed to get drop pipelines from db: %v", err)}
	}

	if len(pipelines) == 0 {
		return pipelines, nil
	}

	for _, d := range pipelines {
		if err := d.ParseRawConfig(); err != nil {
			errors = append(errors, err)
		}
	}

	return pipelines, errors
}

// GetPipelines returns pipeline and errors (if any)
func (r *Repo) GetPipeline(ctx context.Context, id string) (*model.Pipeline, *model.ApiError) {
	pipelines := []model.Pipeline{}

	pipelineQuery := `SELECT id, 
		name, 
		config_json, 
		deployment_status, 
		deployment_sequence  
		FROM pipelines 
		WHERE id = $1`

	err := r.db.SelectContext(ctx, &pipelines, pipelineQuery, id)
	if err != nil {
		zap.S().Errorf("failed to get ingestion pipeline from db", err)
		return nil, model.BadRequestStr("failed to get ingestion pipeline from db")
	}

	if len(pipelines) == 0 {
		zap.S().Warnf("No row found for ingestion pipeline id", id)
		return nil, nil
	}

	if len(pipelines) == 1 {
		err := pipelines[0].ParseRawConfig()
		if err != nil {
			zap.S().Errorf("invalid pipeline config found", id, err)
			return &pipelines[0], model.InternalErrorStr("found an invalid pipeline config ")
		}
		return &pipelines[0], nil
	}

	return nil, model.InternalErrorStr("mutliple pipelines with same id")

}

func (r *Repo) DeletePipeline(ctx context.Context, id string) *model.ApiError {
	deleteQuery := `DELETE
		FROM pipelines 
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, deleteQuery, id)
	if err != nil {
		return model.BadRequest(err)
	}

	return nil

}
