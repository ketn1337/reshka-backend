# Contributing

Спасибо, что участвуете в развитии `reshka-backend`. Этот документ описывает
Git-процесс, соглашение по коммитам и локальные проверки для проекта.

## 1. Веточная модель

Используются две долгоживущие ветки и набор короткоживущих рабочих веток:

| Ветка           | Назначение                                              |
| --------------- | ------------------------------------------------------- |
| `main`          | Стабильная production-ready ветка. Только через PR.     |
| `dev`           | Интеграционная ветка разработки. Только через PR.       |
| `feature/<name>` | Новая функциональность.                                |
| `fix/<name>`     | Исправление бага.                                       |
| `chore/<name>`   | Инструменты, зависимости, мелкая настройка.            |
| `refactor/<name>` | Рефакторинг без изменения поведения.                  |
| `ci/<name>`      | CI/CD, GitHub Actions, branch protection.               |
| `docs/<name>`    | Документация.                                           |

Правило: **одна задача — одна ветка — один PR — удаление ветки после merge**.

## 2. Как начать новую задачу

1. Выберите тип задачи: `feature`, `fix`, `chore`, `refactor`, `ci`, `docs`.
2. Обновите локальную `dev`:

   ```bash
   git checkout dev
   git pull origin dev
   ```

3. Создайте ветку от свежего `dev`:

   ```bash
   git checkout -b feature/short-kebab-name
   ```

> Production hotfix создаётся от `main`, после чего открывается PR
> `fix/...` → `main`, а затем `main` → `dev` для синхронизации.

## 3. Как назвать ветку

Имя строится из типа и краткого описания в kebab-case:

```text
feature/booking-grid-scroll
fix/room-image-paths
chore/bump-gin
refactor/split-booking-service
ci/add-workflow
docs/contributing-guide
```

Используйте только латиницу, цифры и дефис. Без пробелов, без CamelCase,
без указания номера задачи в имени (это упрощает историю).

## 4. Как писать коммиты (Conventional Commits)

Формат первой строки:

```text
<type>(<optional-scope>): <subject>
```

Допустимые типы:

| Type       | Назначение                                              |
| ---------- | ------------------------------------------------------- |
| `feat`     | Новая функциональность.                                 |
| `fix`      | Исправление бага.                                       |
| `chore`    | Инструменты, зависимости, мелкая настройка.             |
| `docs`     | Документация.                                           |
| `refactor` | Рефакторинг без изменения поведения.                    |
| `style`    | Только форматирование, без изменения логики.            |
| `test`     | Только тесты.                                           |
| `ci`       | CI/CD, GitHub Actions.                                  |
| `build`    | Система сборки, скрипты.                                |
| `perf`     | Улучшение производительности.                           |
| `revert`   | Откат ранее сделанного коммита.                         |

Правила:

- `subject` в нижнем регистре, без точки в конце, не длиннее 100 символов.
- Если изменение ломает обратную совместимость, добавьте `!` после типа/скоупа.
- Сообщение пишите на английском языке (короткий summary + развёрнутое описание
  в теле при необходимости).

Примеры:

```text
feat: add booking grid scroll
fix(auth): correct jwt expiration handling
chore: bump gin to v1.10.0
ci: add github actions workflow
docs: update contributing guide
refactor: split booking service
```

Merge, revert, fixup и squash-коммиты, созданные самим Git, проверку
проходят автоматически и не блокируются.

## 5. Локальные проверки

Перед коммитом и пушем выполните:

```bash
make fmt   # gofmt -s -w .
make vet   # go vet ./...
make test  # go test ./...
```

Опционально — все проверки одной командой:

```bash
make lint  # gofmt -l . + go vet ./...
```

Те же шаги выполняет `pre-commit` хук, см. раздел ниже.

## 6. Pre-commit и commit-msg хуки

В репозитории лежат готовые локальные хуки в каталоге `.githooks/`. Они
автоматически активируются после выполнения:

```bash
make hooks
```

Команда делает `git config core.hooksPath .githooks` — после этого Git
будет вызывать скрипты из репозитория, а не из `.git/hooks/`.

- `pre-commit` запускает `gofmt -l`, `go vet ./...` и `go test ./...`
  только если в индексе есть Go-файлы. Чтобы временно пропустить тесты:

  ```bash
  SKIP_PRECOMMIT_TESTS=1 git commit -m "..."
  ```

- `commit-msg` проверяет, что первая строка коммита соответствует
  Conventional Commits. Проверка пропускает merge / revert / fixup /
  squash коммиты, которые создаёт сам Git.

## 7. Как создать PR в `dev`

После локальных коммитов:

```bash
git status
git push -u origin feature/short-kebab-name
```

Затем откройте Pull Request на GitHub:

```text
base:    dev
compare: feature/short-kebab-name
```

PR должен проходить CI (gofmt / vet / test / build) и иметь заполненный
чек-лист из шаблона.

## 8. Release PR из `dev` в `main`

Когда `dev` готов к релизу:

1. Убедитесь, что CI на `dev` зелёный, а в ветке нет временного/отладочного
   кода, TODO-комментариев и закоммиченных секретов.
2. Откройте PR:

   ```text
   base:    main
   compare: dev
   title:   chore: release dev to main
   ```

3. После мержа в `main` и публикации релиза — синхронизируйте `dev` обратно
   (`main` → `dev`), чтобы `dev` не отставал.

## 9. Почему нельзя пушить напрямую в `main` и `dev`

- Прямой push в `main` обходит code review и CI и публикует непроверенный код.
- Прямой push в `dev` ломает идею интеграционной ветки как места, где
  сходится несколько задач сразу — любая ошибка заливает чужие изменения.
- Все изменения должны попадать в долгоживущие ветки **только через PR**,
  чтобы проходить review и CI как единое целое.

Включите branch protection в настройках репозитория на GitHub:

- `main` — запретить прямой push, требовать PR + 1 ревью + зелёный CI.
- `dev` — запретить прямой push, требовать PR + зелёный CI.

## 10. Удаление feature-ветки после merge

GitHub обычно удаляет head-ветку автоматически. Локально:

```bash
git checkout dev
git pull origin dev
git branch -d feature/short-kebab-name
```

Если Git отказывается удалять ветку (unmerged commits), значит ветка не
попала в `dev`. Сначала разберитесь, в каком PR она должна была быть
смёржена, и только потом используйте `git branch -D` как осознанное
принудительное удаление.

Удаление remote-ветки вручную (если GitHub не сделал это сам):

```bash
git push origin --delete feature/short-kebab-name
```

## Команды именно для этого проекта

```bash
# один раз после клонирования
make hooks

# перед каждым коммитом
make fmt
make vet
make test

# запуск API локально
make up        # docker compose up -d  (Postgres)
make migrate   # применить миграции
make run       # go run ./cmd/server

# сборка бинарников
make build
```

## Сводка рабочего потока

```text
feature/*  →  PR  →  dev  →  PR  →  main
   ↑                       ↑
  commit                  release
```
