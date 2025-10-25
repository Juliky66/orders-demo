package app

import (
	"context"
	"encoding/json"
	"log"

	"order/internal/bus"
	"order/internal/cache"
	"order/internal/model"
	"order/internal/storage"
)

type App struct {
	Store storage.Store
	Cache cache.Cache
	Sub   bus.Subscriber
}

func (a *App) RestoreCache(ctx context.Context) error {
	orders, err := a.Store.LoadAllRaw(ctx)
	if err != nil {
		return err
	}
	for _, o := range orders {
		a.Cache.Set(o.OrderUID, o)
	}
	log.Printf("cache restored (%d)", len(orders))
	return nil
}

func (a *App) HandleMsg(data []byte) error {
	var o model.Order
	if err := json.Unmarshal(data, &o); err != nil {
		return err
	}
	if o.OrderUID == "" {
		return ErrBadOrder
	}
	if err := a.Store.UpsertOrder(context.Background(), o); err != nil {
		return err
	}
	a.Cache.Set(o.OrderUID, o)
	log.Printf("order upserted: %s (items=%d)", o.OrderUID, len(o.Items))
	return nil
}

var ErrBadOrder = &BadOrderError{}

type BadOrderError struct{}

func (e *BadOrderError) Error() string { return "bad order" }
