// Package controller holds the process entry points: the WAL listener and the HTTP
// servers, each binding a transport to handlers built in the application layer.
package controller

import (
	"context"
	"hump/internal/application/processing/process_operation"
	"hump/internal/port/database"
)

// ProcessController feeds the WAL stream of db into the process_operation handler.
type ProcessController struct {
	db         database.Database
	controller *process_operation.Controller
}

// NewProcessController returns a ProcessController that hands every decoded row
// from db to controller.
func NewProcessController(db database.Database, controller *process_operation.Controller) *ProcessController {
	return &ProcessController{
		db:         db,
		controller: controller,
	}
}

// Listen streams WAL changes into the handler until ctx is cancelled or the
// stream fails.
func (c *ProcessController) Listen(ctx context.Context) error {
	return c.db.Listen(ctx, c.controller.Process)
}
