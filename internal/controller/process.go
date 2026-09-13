package controller

import (
	"context"
	"hump/internal/application/processing/process_operation"
	"hump/internal/port/database"
)

type ProcessController struct {
	db         database.Database
	controller *process_operation.Controller
}

func NewProcessController(db database.Database, controller *process_operation.Controller) *ProcessController {
	return &ProcessController{
		db:         db,
		controller: controller,
	}
}

func (c *ProcessController) Listen(ctx context.Context) error {
	return c.db.Listen(ctx, c.controller.Process)
}
