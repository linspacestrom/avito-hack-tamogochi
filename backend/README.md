# Avito Tamagotchi Backend

Backend игрового сервиса «Авито Тамагочи». API написан на Go, использует PostgreSQL,
`chi`, `pgx`, `sqlc`, JWT-авторизацию и Swagger UI.

## Требования

- Go 1.25+
- PostgreSQL
- `golang-migrate` для применения миграций
- `sqlc` для повторной генерации кода запросов
- `swag` для обновления Swagger-документации

Установить вспомогательные CLI:

```bash
go install -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate@latest
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
go install github.com/swaggo/swag/cmd/swag@latest
```

CLI устанавливаются через `go install`, а не через `go get`: тогда они не попадают в
зависимости приложения.

## Быстрый запуск

Все команды ниже выполняются из директории `backend`.

### 1. Скачать зависимости

```bash
go mod download
```

### 2. Настроить окружение

Создайте файл `.env` в директории `backend`:

```dotenv
APP_NAME=avito-tamagotchi
APP_ENV=development

HTTP_HOST=0.0.0.0
HTTP_PORT=8080

LOG_LEVEL=debug
LOG_FORMAT=text

DB_HOST=localhost
DB_PORT=5432
DB_USERNAME=postgres
DB_PASSWORD=your_password
DB_NAME=avito_tamagotchi
DB_SSLMODE=disable

ACCESS_TOKEN_TTL=15m
REFRESH_TOKEN_TTL=720h
AUTH_SECRET=replace_with_your_own_secret

# Используется только CLI golang-migrate, приложение эту переменную не читает.
DATABASE_URL=postgres://postgres:your_password@localhost:5432/avito_tamagotchi?sslmode=disable
```

`.env` находится в `.gitignore` и не должен попадать в репозиторий.

### 3. Сгенерировать секрет авторизации

`AUTH_SECRET` обязателен и должен содержать не меньше 32 байт. Сгенерируйте его сами:

```bash
openssl rand -hex 32
```

Команда вернёт 64 шестнадцатеричных символа. Скопируйте результат в `AUTH_SECRET`.
Не используйте секреты из README, исходного кода или чужого окружения.

Для одноразового запуска без записи секрета в файл:

```bash
AUTH_SECRET="$(openssl rand -hex 32)" go run ./cmd/server
```

### 4. Применить миграции

Загрузить переменные из `.env` в текущий shell:

```bash
set -a
source .env
set +a
```

Применить все миграции:

```bash
migrate -path migrations -database "$DATABASE_URL" up
```

Проверить текущую версию:

```bash
migrate -path migrations -database "$DATABASE_URL" version
```

Откатить одну миграцию:

```bash
migrate -path migrations -database "$DATABASE_URL" down 1
```

Создать новую пару миграций:

```bash
migrate create -ext sql -dir migrations -seq migration_name
```

Будут созданы файлы:

```text
migrations/000002_migration_name.up.sql
migrations/000002_migration_name.down.sql
```

Не изменяйте уже применённую миграцию. Создавайте следующую.

### 5. Запустить сервер

```bash
go run ./cmd/server
```

По умолчанию API доступен на `http://localhost:8080`.

## Конфигурация

Основной YAML-файл расположен в `config/config.yaml`. Другой файл можно выбрать через:

```bash
CONFIG_PATH=/path/to/config.yaml go run ./cmd/server
```

Несекретные настройки можно хранить в YAML:

```yaml
http:
  host: "0.0.0.0"
  port: "8080"
  read_timeout: 5s
  write_timeout: 5s
  idle_timeout: 60s
  shutdown_timeout: 5s

database:
  host: "localhost"
  port: "5432"
  username: "postgres"
  password: "local_password"
  name: "avito_tamagotchi"
  ssl_mode: "disable"
  max_open_conns: 20
  min_open_conns: 5
  conn_max_lifetime: 1h
  conn_max_idle_time: 25m
  timeout: 5s

auth:
  access_token_ttl: 15m
  refresh_token_ttl: 720h
```

Переменные окружения имеют приоритет над YAML. Локально их можно записать в `.env`.

`AUTH_SECRET` намеренно нельзя задать через YAML: он читается только из переменной
окружения или `.env`. Пароли также рекомендуется хранить в `.env`, а не коммитить в YAML.

Основные переменные:

- `HTTP_HOST`, `HTTP_PORT`
- `LOG_LEVEL`, `LOG_FORMAT`
- `DB_HOST`, `DB_PORT`, `DB_USERNAME`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE`
- `ACCESS_TOKEN_TTL`, `REFRESH_TOKEN_TTL`
- `AUTH_SECRET`
- `CONFIG_PATH`

## Swagger

После запуска интерактивная документация доступна по адресу:

```text
http://localhost:8080/swagger/index.html
```

Через Swagger UI можно выполнить регистрацию и вход. Для защищённых запросов:

1. Получите `accessToken` через `/api/auth/register` или `/api/auth/login`.
2. Нажмите `Authorize`.
3. Введите `Bearer <accessToken>`.
4. Выполните запрос `/api/auth/me`.

Swagger генерируется из комментариев над HTTP handlers. После изменения маршрутов, DTO
или аннотаций обновите документацию:

```bash
"$(go env GOPATH)/bin/swag" init \
  -g cmd/server/main.go \
  -o docs \
  --parseInternal
```

Файлы в `docs/` сгенерированы автоматически. Не редактируйте их вручную.

## sqlc

SQL-запросы находятся в:

```text
internal/repository/postgres/query/
```

Конфигурация генератора — `sqlc.yaml`. После изменения SQL-запросов или схемы:

```bash
sqlc generate
```

Если `sqlc` отсутствует в `PATH`:

```bash
"$(go env GOPATH)/bin/sqlc" generate
```

Сгенерированный код находится в `internal/repository/postgres/sqlc/`. Его нельзя
редактировать вручную.

`sqlc` не применяет миграции. Он только проверяет SQL и генерирует типобезопасный Go-код.
Схему реальной БД изменяет `golang-migrate`.

## Авторизация

- Access-токен — JWT в заголовке `Authorization: Bearer <token>`.
- Access TTL по умолчанию — 15 минут.
- Refresh-токен передаётся в JSON и хранится в БД только как SHA-256 hash.
- Refresh TTL по умолчанию — 30 дней.
- При refresh старая refresh-сессия отзывается и создаётся новая.
- После refresh или logout ранее выданный access JWT действует до окончания своего TTL.
- Cookies не используются.

Доступные endpoints:

- `POST /api/auth/register`
- `POST /api/auth/login`
- `POST /api/auth/refresh`
- `POST /api/auth/logout`
- `GET /api/auth/me`

## Структура проекта

```text
backend/
├── cmd/server/                     # точка входа
├── config/                         # YAML-конфигурация
├── docs/                           # generated Swagger
├── migrations/                     # up/down SQL-миграции
├── internal/
│   ├── app/                        # composition root, сборка зависимостей
│   ├── apperror/                   # общие ошибки приложения
│   ├── config/                     # чтение YAML и environment
│   ├── dto/                        # HTTP request/response модели
│   ├── entity/                     # доменные сущности и инварианты
│   ├── handler/                    # HTTP handlers
│   ├── logger/                     # slog
│   ├── mapper/                     # entity → DTO
│   ├── middleware/                 # Bearer auth и request logging
│   ├── repository/postgres/        # PostgreSQL adapters и sqlc
│   ├── router/                     # chi routes и Swagger UI
│   ├── server/                     # lifecycle HTTP-сервера
│   ├── service/                    # use cases
│   └── token/                      # access/refresh token manager
├── go.mod
├── go.sum
└── sqlc.yaml
```

Поток HTTP-запроса:

```text
client → chi router → middleware → handler → service → repository → PostgreSQL
```

`internal/app` является composition root: только этот пакет знает конкретные реализации
репозиториев, сервисов, token manager и HTTP transport.

## Полезные команды

Форматирование:

```bash
go fmt ./...
```

Проверка компиляции пакетов:

```bash
go test ./...
```

Статический анализ:

```bash
go vet ./...
```

Сборка:

```bash
go build -o bin/server ./cmd/server
```

Очистка зависимостей:

```bash
go mod tidy
```

`go mod tidy` запускайте после добавления или удаления импортов. Не используйте `go get`
для установки CLI-инструментов.

## Важные замечания

- Перед первым запуском PostgreSQL должен быть доступен: приложение проверяет соединение
  при старте.
- Миграции применяются отдельно и автоматически при запуске приложения не выполняются.
- После изменения миграций/queries перегенерируйте sqlc.
- После изменения API-аннотаций перегенерируйте Swagger.
- Не коммитьте `.env`, реальные пароли, `AUTH_SECRET` и `DATABASE_URL`.
