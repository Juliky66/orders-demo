package storage

import (
	"context"

	"order/internal/model"
)

type Store interface {
	UpsertOrder(ctx context.Context, o model.Order) error
	LoadAllRaw(ctx context.Context) ([]model.Order, error)
	Close()
}
