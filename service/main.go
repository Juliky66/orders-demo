package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	stan "github.com/nats-io/stan.go"
)

type Delivery struct {
	Name    string `json:"name"`
	Phone   string `json:"phone"`
	ZIP     string `json:"zip"`
	City    string `json:"city"`
	Address string `json:"address"`
	Region  string `json:"region"`
	Email   string `json:"email"`
}

type Payment struct {
	Transaction  string `json:"transaction"`
	RequestID    string `json:"request_id"`
	Currency     string `json:"currency"`
	Provider     string `json:"provider"`
	Amount       int    `json:"amount"`
	PaymentDT    int64  `json:"payment_dt"`
	Bank         string `json:"bank"`
	DeliveryCost int    `json:"delivery_cost"`
	GoodsTotal   int    `json:"goods_total"`
	CustomFee    int    `json:"custom_fee"`
}

type Item struct {
	ChrtID      int64  `json:"chrt_id"`
	TrackNumber string `json:"track_number"`
	Price       int    `json:"price"`
	RID         string `json:"rid"`
	Name        string `json:"name"`
	Sale        int    `json:"sale"`
	Size        string `json:"size"`
	TotalPrice  int    `json:"total_price"`
	NMID        int64  `json:"nm_id"`
	Brand       string `json:"brand"`
	Status      int    `json:"status"`
}

type Order struct {
	OrderUID          string    `json:"order_uid"`
	TrackNumber       string    `json:"track_number"`
	Entry             string    `json:"entry"`
	Delivery          Delivery  `json:"delivery"`
	Payment           Payment   `json:"payment"`
	Items             []Item    `json:"items"`
	Locale            string    `json:"locale"`
	InternalSignature string    `json:"internal_signature"`
	CustomerID        string    `json:"customer_id"`
	DeliveryService   string    `json:"delivery_service"`
	ShardKey          string    `json:"shardkey"`
	SmID              int       `json:"sm_id"`
	DateCreated       time.Time `json:"date_created"`
	OofShard          string    `json:"oof_shard"`
}

type App struct {
	db    *pgxpool.Pool
	cache sync.Map // key: order_uid, value: Order
}

func mustEnv(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

func main() {
	// --- env ---
	pgURL := mustEnv("PG_URL", "postgres://demo:demo@localhost:5432/ordersdb")
	stanClusterID := mustEnv("STAN_CLUSTER", "orders-cluster")
	stanClientID := mustEnv("STAN_CLIENT", "orders-service")
	stanURL := mustEnv("STAN_URL", "nats://localhost:4222")
	subject := mustEnv("STAN_SUBJECT", "orders")
	httpAddr := mustEnv("HTTP_ADDR", ":8080")

	// --- DB pool ---
	ctx := context.Background()
	dbpool, err := pgxpool.New(ctx, pgURL)
	if err != nil {
		log.Fatal("pg connect:", err)
	}
	defer dbpool.Close()

	app := &App{db: dbpool}

	// --- восстановление кэша из БД при старте ---
	if err := app.restoreCache(ctx); err != nil {
		log.Fatal("restore cache:", err)
	}
	log.Println("cache restored")

	// --- NATS-Streaming conn ---
	sc, err := stan.Connect(stanClusterID, stanClientID, stan.NatsURL(stanURL))
	if err != nil {
		log.Fatal("stan connect:", err)
	}
	defer sc.Close()

	// Durable + Manual Ack, чтобы не терять сообщения
	_, err = sc.Subscribe(subject, func(m *stan.Msg) {
		if err := app.handleMessage(ctx, m.Data); err != nil {
			log.Println("handle msg error:", err)
			// не ack — сообщение будет доставлено снова
			return
		}
		if err := m.Ack(); err != nil {
			log.Println("ack error:", err)
		}
	}, stan.DurableName("orders-durable"), stan.SetManualAckMode(), stan.DeliverAllAvailable())
	if err != nil {
		log.Fatal("subscribe:", err)
	}

	// --- HTTP ---
	mux := http.NewServeMux()
	mux.HandleFunc("/orders/", app.getOrderByID)
	mux.Handle("/", http.FileServer(http.Dir("./static"))) // простейший UI

	srv := &http.Server{Addr: httpAddr, Handler: mux}
	go func() {
		log.Println("http listen", httpAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	// --- graceful shutdown ---
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Println("shutting down...")
	_ = srv.Shutdown(context.Background())
}

// Восстановление кэша из БД
func (a *App) restoreCache(ctx context.Context) error {
	rows, err := a.db.Query(ctx, `select raw from orders`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return err
		}
		var ord Order
		if err := json.Unmarshal(raw, &ord); err != nil {
			return err
		}
		a.cache.Store(ord.OrderUID, ord)
	}
	return rows.Err()
}

// Обработка входящего JSON: валидация, апсерт в БД, обновление кэша
func (a *App) handleMessage(ctx context.Context, data []byte) error {
	var ord Order
	if err := json.Unmarshal(data, &ord); err != nil {
		return err // мусор — отклоняем
	}
	if ord.OrderUID == "" {
		return errors.New("missing order_uid")
	}

	// транзакция: апсерт orders + delivery + payment + items
	tx, err := a.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// orders
	if _, err := tx.Exec(ctx, `
insert into orders(order_uid,track_number,entry,locale,internal_signature,customer_id,
                   delivery_service,shardkey,sm_id,date_created,oof_shard,raw)
values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
on conflict (order_uid) do update set
  track_number=excluded.track_number,
  entry=excluded.entry,
  locale=excluded.locale,
  internal_signature=excluded.internal_signature,
  customer_id=excluded.customer_id,
  delivery_service=excluded.delivery_service,
  shardkey=excluded.shardkey,
  sm_id=excluded.sm_id,
  date_created=excluded.date_created,
  oof_shard=excluded.oof_shard,
  raw=excluded.raw
`, ord.OrderUID, ord.TrackNumber, ord.Entry, ord.Locale, ord.InternalSignature, ord.CustomerID,
		ord.DeliveryService, ord.ShardKey, ord.SmID, ord.DateCreated, ord.OofShard, data); err != nil {
		return err
	}

	// delivery
	if _, err := tx.Exec(ctx, `
insert into delivery(order_uid,name,phone,zip,city,address,region,email)
values ($1,$2,$3,$4,$5,$6,$7,$8)
on conflict (order_uid) do update set
  name=excluded.name, phone=excluded.phone, zip=excluded.zip, city=excluded.city,
  address=excluded.address, region=excluded.region, email=excluded.email
`, ord.OrderUID, ord.Delivery.Name, ord.Delivery.Phone, ord.Delivery.ZIP, ord.Delivery.City,
		ord.Delivery.Address, ord.Delivery.Region, ord.Delivery.Email); err != nil {
		return err
	}

	// payment
	if _, err := tx.Exec(ctx, `
insert into payment(order_uid,transaction,request_id,currency,provider,amount,payment_dt,bank,delivery_cost,goods_total,custom_fee)
values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
on conflict (order_uid) do update set
  transaction=excluded.transaction, request_id=excluded.request_id, currency=excluded.currency,
  provider=excluded.provider, amount=excluded.amount, payment_dt=excluded.payment_dt, bank=excluded.bank,
  delivery_cost=excluded.delivery_cost, goods_total=excluded.goods_total, custom_fee=excluded.custom_fee
`, ord.OrderUID, ord.Payment.Transaction, ord.Payment.RequestID, ord.Payment.Currency, ord.Payment.Provider,
		ord.Payment.Amount, ord.Payment.PaymentDT, ord.Payment.Bank, ord.Payment.DeliveryCost, ord.Payment.GoodsTotal, ord.Payment.CustomFee); err != nil {
		return err
	}

	// items: для простоты удалим и вставим заново
	if _, err := tx.Exec(ctx, `delete from items where order_uid=$1`, ord.OrderUID); err != nil {
		return err
	}
	for _, it := range ord.Items {
		_, err := tx.Exec(ctx, `
insert into items(order_uid,chrt_id,track_number,price,rid,name,sale,size,total_price,nm_id,brand,status)
values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
`, ord.OrderUID, it.ChrtID, it.TrackNumber, it.Price, it.RID, it.Name, it.Sale, it.Size, it.TotalPrice, it.NMID, it.Brand, it.Status)
		if err != nil {
			return err
		}
	}
	// commit
	if err := tx.Commit(ctx); err != nil {
		return err
	}

	// обновить кэш
	a.cache.Store(ord.OrderUID, ord)

	// Вывести order_uid в консоль для удобства
	log.Printf("Получен и сохранён заказ: %s", ord.OrderUID)
	return nil
}

// HTTP: GET /orders/{id}
func (a *App) getOrderByID(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/orders/"):]
	if id == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}
	if v, ok := a.cache.Load(id); ok {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(v)
		return
	}
	http.NotFound(w, r)
}
