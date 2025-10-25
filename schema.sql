-- Создание пользователя и базы данных

CREATE USER demo WITH PASSWORD 'demo';
CREATE DATABASE ordersdb OWNER demo;

-- Подключаемся к базе
\c ordersdb demo;

-- Схема таблиц для сервиса Orders Demo

-- Таблица заказов (основная)
CREATE TABLE IF NOT EXISTS orders (
  order_uid TEXT PRIMARY KEY,
  track_number TEXT,
  entry TEXT,
  locale TEXT,
  internal_signature TEXT,
  customer_id TEXT,
  delivery_service TEXT,
  shardkey TEXT,
  sm_id INTEGER,
  date_created TIMESTAMPTZ,
  oof_shard TEXT,
  raw JSONB NOT NULL
);

-- Таблица доставки (1:1 с orders)
CREATE TABLE IF NOT EXISTS delivery (
  order_uid TEXT PRIMARY KEY REFERENCES orders(order_uid) ON DELETE CASCADE,
  name TEXT,
  phone TEXT,
  zip TEXT,
  city TEXT,
  address TEXT,
  region TEXT,
  email TEXT
);

-- Таблица оплаты (1:1 с orders)
CREATE TABLE IF NOT EXISTS payment (
  order_uid TEXT PRIMARY KEY REFERENCES orders(order_uid) ON DELETE CASCADE,
  transaction TEXT,
  request_id TEXT,
  currency TEXT,
  provider TEXT,
  amount INTEGER,
  payment_dt BIGINT,
  bank TEXT,
  delivery_cost INTEGER,
  goods_total INTEGER,
  custom_fee INTEGER
);

-- Таблица товаров (1:N с orders)
CREATE TABLE IF NOT EXISTS items (
  id BIGSERIAL PRIMARY KEY,
  order_uid TEXT REFERENCES orders(order_uid) ON DELETE CASCADE,
  chrt_id BIGINT,
  track_number TEXT,
  price INTEGER,
  rid TEXT,
  name TEXT,
  sale INTEGER,
  size TEXT,
  total_price INTEGER,
  nm_id BIGINT,
  brand TEXT,
  status INTEGER
);

-- Индекс для ускорения выборки по order_uid
CREATE INDEX IF NOT EXISTS idx_items_order_uid ON items(order_uid);

