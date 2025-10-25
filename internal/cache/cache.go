package cache

import "order/internal/model"

type Cache interface {
	Set(id string, o model.Order)
	Get(id string) (model.Order, bool)
	Range(func(string, model.Order) bool)
}
