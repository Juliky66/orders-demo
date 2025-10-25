
# NATS Streaming 

Демонстрационный сервис для отображения данных о заказах.

## Стэк

* Go 
* PostgreSQL
* NATS Streaming Server 

## Функционал

* Подключается к локальному серверу **NATS Streaming** и подписывается на канал `orders`
* Принимает заказы в формате JSON (`model.json`, `model2.json`)
* Сохраняет заказы в **PostgreSQL**
* Кэширует их в памяти (`in-memory`)
* При перезапуске восстанавливает кэш из БД
* HTTP-сервер на `localhost:8080`

## Как запустить проект 

### 1. Настройка базы данных PostgreSQL

#### Создание пользователя и базы данных

Открыть консоль `psql` (или PowerShell в каталоге PostgreSQL, например
`C:\Program Files\PostgreSQL\18\bin`) и выполнить:

```sql
CREATE USER demo WITH PASSWORD 'demo';
CREATE DATABASE ordersdb OWNER demo;
```

#### Применение схемы

В корне проекта (`order`) находится файл **`schema.sql`**, который создаёт все таблицы и связи.
Чтобы применить его, выполнить:

```bash
psql -U postgres -h localhost -f "C:\Users\1645305\Desktop\order\schema.sql"
```

*Замечание*: Если путь отличается - подставить свой. Можно все выполнить в среде PgAdmin.

#### Проверка

Подключись под пользователем `demo`:

```bash
psql -U demo -d ordersdb
```

### 2. Запуск NATS Streaming Server

В папке, где находится `nats-streaming-server.exe` выполнить:

```bash
nats-streaming-server.exe -p 4222 -m 8222 -cid orders-cluster --store file --dir .\data
```

### 3. Запуск сервиса

Открыть новое окно CMD или терминал в VS Code и выполнить:

```bash
set PG_URL=postgres://demo:demo@localhost:5432/ordersdb
set STAN_CLUSTER=orders-cluster
set STAN_CLIENT=orders-service
set STAN_URL=nats://localhost:4222
set STAN_SUBJECT=orders
set HTTP_ADDR=:8080

go run ./go/service
```

### 4. Отправка тестовых заказов

Открыть второе окно CMD или терминал в VS Code:

```bash
go run .\go\publisher .\model.json 
go run .\go\publisher .\model2.json 
```

После публикации сервис получит заказы:

```
order upserted: b563feb7b2b84b6test (items=1)
order upserted: a987xyzt4321example (items=2)
```

### 5. Проверить результаты

Перейди в браузере по адресу:
[http://localhost:8080](http://localhost:8080)

## Видео демонстрация

[Смотреть видео на Google Drive](https://drive.google.com/...)
