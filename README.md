# Mood News Grid

Приложение для отображения новостей с возможностью классификации/генерации сопроводительных фактов.

Backend на Go, frontend на TypeScript + Vite. Подойдёт для демонстрации сеточной раскладки новостей и интеграции с внешними LLM/AI-сервисами (есть код в `backend/openrouter`).

## Стек
- Языки: Go (backend), TypeScript (frontend)
- Рантаймы/фреймворки:
  - Backend: Go (модули через go.mod)
  - Frontend: Vite (конфигурация в `frontend/vite.config.ts`)
- Основные папки:
  - `backend/` — серверная часть на Go (точка входа `backend/main.go`)
  - `frontend/` — клиентская часть (Vite, `package.json`, `src/`)
  - `backend/db/` — схема и работа с базой (`schema.sql`, `db.go`)

## Что умеет
- HTTP API для работы с новостями (обработчики в `backend/handler/`)
- SSE (Server-Sent Events) для потокового обновления (`backend/handler/sse.go`)
- Механизмы валидации и аутентификации (`backend/validator`, `backend/middleware/auth.go`)
- Интеграция с внешним маршрутизатором/AI (`backend/openrouter`)

## Как запустить (локально)
Самый короткий путь — использовать Docker Compose или запустить backend и frontend по отдельности.

Запуск через Docker Compose (если установлен Docker):

```bash
# из корня репозитория
docker compose -f backend/compose.yaml up --build
```

Запуск backend локально (Go 1.20+):

```bash
# из корня репозитория
cd backend
go mod download
# Запуск в режиме разработки
go run ./main.go
# или собрать бинарник
go build -o mood-backend ./
./mood-backend
```

Запуск frontend (Node/Bun + Vite):

```bash
cd frontend
# установка зависимостей (npm, pnpm или bun)
npm install
# или: bun install
# затем
npm run dev
# или: bun run dev
```

## Переменные окружения
В репозитории нет единого .env, но backend может требовать настройки (напр., адрес базы данных, ключи для интеграции с внешними сервисами). Проверьте в коде `backend/main.go` и `backend/utils` какие переменные ожидаются.

## Структура репозитория (корневые папки)
```
backend/        # Go-сервер: main.go, handler/, middleware/, db/, openrouter/
frontend/       # Веб-клиент: vite + TypeScript, src/, public/
```

## Вклад и разработка
- Открывайте PR для новых фич и багфиксов.
- Пишите тесты и обновляйте README по мере изменения команд запуска.

---

Автор: ruslan-kuchaev
