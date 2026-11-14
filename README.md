# Посты и комментарии

GraphQL-сервис на Go с хранилищем в памяти или PostgreSQL. Поддерживает вложенные комментарии, пагинацию и запрет новых комментариев автором поста.

## Запуск

Нужны Docker и Docker Compose v2:

```sh
cp .env.example .env
```

Далее надо выбрать один из режимов:

```sh
docker compose --profile memory up --build app-memory
```

```sh
docker compose --profile postgres up --build app-postgres
```

Playground: <http://localhost:8080/>. API: `POST /query`.

Остановка: `docker compose --profile memory --profile postgres down`.

Для локального запуска нужен Go 1.26+:

```sh
export AUTH_USERS='[{"id":1,"name":"Alice","email":"alice@example.com","token":"alice-token"}]'
STORAGE=memory go run ./cmd/server
```

Для PostgreSQL нужно задать `DATABASE_URL`, выполнить `go run ./cmd/migrate`, затем `STORAGE=postgres go run ./cmd/server`.

## Настройки

| Переменная | Назначение / значение по умолчанию |
| --- | --- |
| `STORAGE` | `memory` (по умолчанию) или `postgres` |
| `HTTP_ADDR` | Адрес сервера, `:8080` |
| `DATABASE_URL` | Строка подключения, обязательна для PostgreSQL |
| `AUTH_USERS` | Обязательный JSON-массив `{id,name,email,token}` |
| `HTTP_PORT` | Порт на хосте в Compose, `8080` |
| `POSTGRES_PASSWORD` | Пароль БД в Compose, `development-password` |

Пользователи создаются на старте из `AUTH_USERS`.

## Проверки и генерация

```sh
make test
make vet
make build
make generate
```

Тесты внутри Docker: `docker build --target test .`.

## Структура

```text
cmd/server                    сборка зависимостей и HTTP-сервер
cmd/migrate                   запуск миграций
db/migrations                 схема БД и мигратор
internal/domain               общие сущности, ошибки и параметры чтения
internal/service
  posts, comments, users      бизнес-логика и интерфейсы репозиториев
  validation                  общая валидация
internal/repository
  memory
    posts, comments, users    операции над сущностями и тесты
    internal/store            общее состояние и мьютексы
    internal/pagination       бинарный поиск по индексам
  postgres
    posts, comments, users    SQL, обёртки репозиториев и тесты
    dbgen                     сгенерированный sqlc-код
    internal/database         перевод ошибок и общие SQL-типы
  cursor                      кодирование и проверка курсоров
internal/transport
  graphql                     HTTP GraphQL-обработчик
    resolver                  резолверы и интерфейсы сервисов
    loader                    batch-загрузка и предзагрузка дерева
    convert                   преобразование домена и ошибок
    model, generated          сгенерированный gqlgen-код
  http/middleware             Bearer-аутентификация
```
