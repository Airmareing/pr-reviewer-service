# PR Reviewer Assignment Service

Микросервис для автоматического назначения ревьюверов на Pull Request'ы.

## Возможности

- Создание команд и управление пользователями
- Автоматическое назначение до 2 ревьюверов при создании PR
- Переназначение ревьюверов из команды заменяемого
- Управление статусами PR (OPEN/MERGED)
- REST API согласно OpenAPI спецификации

## Стек

- Go 1.25.4
- PostgreSQL 15
- Docker & Docker Compose

## Быстрый старт
```bash
docker-compose up --build
```

Сервис доступен на `http://localhost:8080`

## API Endpoints

| Метод | Путь | Описание |
|-------|------|----------|
| POST | `/team/add` | Создать команду с участниками |
| GET | `/team/get?team_name=...` | Получить команду |
| POST | `/users/setIsActive` | Изменить активность пользователя |
| GET | `/users/getReview?user_id=...` | Получить PR пользователя |
| POST | `/pullRequest/create` | Создать PR с автоназначением ревьюверов |
| POST | `/pullRequest/merge` | Merge PR (идемпотентно) |
| POST | `/pullRequest/reassign` | Переназначить ревьювера |
| GET | `/health` | Health check |