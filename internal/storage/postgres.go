package storage

import (
	"context"
	"encoding/json"
	"fmt"

	"order/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct{ db *pgxpool.Pool }

func NewPostgres(ctx context.Context, url string) (*Postgres, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	if err = pool.Ping(ctx); err != nil {
		return nil, err
	}
	return &Postgres{db: pool}, nil
}

func (p *Postgres) Close() { p.db.Close() }

// UpsertOrder сохраняет заказ в orders/delivery/payment и пересобирает items.
func (p *Postgres) UpsertOrder(ctx context.Context, o model.Order) error {
	tx, err := p.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	raw, _ := json.Marshal(o)

	// orders
	_, err = tx.Exec(ctx, `
INSERT INTO orders (
  order_uid, track_number, entry, locale, internal_signature, customer_id,
  delivery_service, shardkey, sm_id, date_created, oof_shard, raw
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
ON CONFLICT (order_uid) DO UPDATE SET
  track_number=EXCLUDED.track_number,
  entry=EXCLUDED.entry,
  locale=EXCLUDED.locale,
  internal_signature=EXCLUDED.internal_signature,
  customer_id=EXCLUDED.customer_id,
  delivery_service=EXCLUDED.delivery_service,
  shardkey=EXCLUDED.shardkey,
  sm_id=EXCLUDED.sm_id,
  date_created=EXCLUDED.date_created,
  oof_shard=EXCLUDED.oof_shard,
  raw=EXCLUDED.raw
`, o.OrderUID, o.TrackNumber, o.Entry, o.Locale, o.InternalSignature, o.CustomerID,
		o.DeliveryService, o.ShardKey, o.SmID, o.DateCreated, o.OofShard, raw)
	if err != nil {
		return fmt.Errorf("upsert orders: %w", err)
	}

	// delivery
	d := o.Delivery
	_, err = tx.Exec(ctx, `
INSERT INTO delivery (
  order_uid, name, phone, zip, city, address, region, email
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
ON CONFLICT (order_uid) DO UPDATE SET
  name=EXCLUDED.name,
  phone=EXCLUDED.phone,
  zip=EXCLUDED.zip,
  city=EXCLUDED.city,
  address=EXCLUDED.address,
  region=EXCLUDED.region,
  email=EXCLUDED.email
`, o.OrderUID, d.Name, d.Phone, d.ZIP, d.City, d.Address, d.Region, d.Email)
	if err != nil {
		return fmt.Errorf("upsert delivery: %w", err)
	}

	// payment
	pay := o.Payment
	_, err = tx.Exec(ctx, `
INSERT INTO payment (
  order_uid, transaction, request_id, currency, provider, amount, payment_dt,
  bank, delivery_cost, goods_total, custom_fee
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
ON CONFLICT (order_uid) DO UPDATE SET
  transaction=EXCLUDED.transaction,
  request_id=EXCLUDED.request_id,
  currency=EXCLUDED.currency,
  provider=EXCLUDED.provider,
  amount=EXCLUDED.amount,
  payment_dt=EXCLUDED.payment_dt,
  bank=EXCLUDED.bank,
  delivery_cost=EXCLUDED.delivery_cost,
  goods_total=EXCLUDED.goods_total,
  custom_fee=EXCLUDED.custom_fee
`, o.OrderUID, pay.Transaction, pay.RequestID, pay.Currency, pay.Provider, pay.Amount,
		pay.PaymentDT, pay.Bank, pay.DeliveryCost, pay.GoodsTotal, pay.CustomFee)
	if err != nil {
		return fmt.Errorf("upsert payment: %w", err)
	}

	// items: пересобираем
	if _, err = tx.Exec(ctx, `DELETE FROM items WHERE order_uid=$1`, o.OrderUID); err != nil {
		return fmt.Errorf("clear items: %w", err)
	}
	for _, it := range o.Items {
		_, err = tx.Exec(ctx, `
INSERT INTO items (
  order_uid, chrt_id, track_number, price, rid, name, sale, size,
  total_price, nm_id, brand, status
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
`, o.OrderUID, it.ChrtID, it.TrackNumber, it.Price, it.RID, it.Name, it.Sale, it.Size,
			it.TotalPrice, it.NMID, it.Brand, it.Status)
		if err != nil {
			return fmt.Errorf("insert item: %w", err)
		}
	}

	return tx.Commit(ctx)
}

// LoadAllRaw достаёт все заказы из orders.raw и распаковывает их в модель.
func (p *Postgres) LoadAllRaw(ctx context.Context) ([]model.Order, error) {
	rows, err := p.db.Query(ctx, `SELECT raw FROM orders`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.Order
	for rows.Next() {
		var b []byte
		if err := rows.Scan(&b); err != nil {
			return nil, err
		}
		var o model.Order
		if err := json.Unmarshal(b, &o); err != nil {
			return nil, fmt.Errorf("bad raw: %w", err)
		}
		out = append(out, o)
	}
	return out, rows.Err()
}
