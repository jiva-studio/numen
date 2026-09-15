# План: архитектурная нормализация, чистка ADR, глоссария и кодовой базы

Этот документ фиксирует утверждённый план полировки архитектуры, устранения противоречий в ADR и глоссарии, чистки устаревшей документации, удаления мёртвых файлов и нормализации нейминга в ветке `feat/a-book-made-for-a-screen-is-read` (PR #270).

---

## 1. Текущий статус и фундамент

Ветка `feat/a-book-made-for-a-screen-is-read` реализует читалку EPUB-книг со свободным перетеканием текста (reflow), колоночной вёрсткой и разворотами (spreads).

Все 17 проверок GitHub Actions успешно проходят:
- Boundaries (FSD-слои, правила зависимостей, отсутствие циклических импортов).
- Core / Lint Go (`golangci-lint`, отсутствие `panic`).
- Core / Test Go Linux (`go test -race`, форматирование `gofmt`).
- Desktop & Mobile / Test Screens & Go.
- Protocol / Check Schema (`buf lint`, кодогенерация `buf generate`).
- Docs / Build (`manual:check`, `astro check`, `eslint`, `astro build`).
- Stories / Check Shots (`manual.mjs`, `stories.mjs`).

---

## 2. Устранение противоречий в ADR (`docs/adr/`)

### 2.1. ADR-0044: Утверждение стандарта форматирования (Prettier)
* **Проблема:** [ADR-0044](docs/adr/0044-typescript-and-vue-are-formatted-by-hand.md) категорически запрещал Prettier (*«Prettier is not installed, and no formatter is run over TypeScript, Vue or Markdown»*). При этом коммит `9093fc89f` внедрил Prettier, плагин для Tailwind и отформатировал 452 файла фронтенда.
* **Решение:**
  - Обновить ADR-0044: переписать статус и решение, зафиксировав официальное принятие Prettier + `prettier-plugin-tailwindcss` для фронтенд-модулей.
  - Закрепить обязательное архитектурное исключение в `.prettierignore` для кодогенерации протокола (`modules/libs/protocol/src/numen/v1/**`), чтобы форматирование не ломало `buf generate`.

### 2.2. ADR-0024: Компонентная библиотека (Reka UI вместо shadcn CLI)
* **Проблема:** [ADR-0024](docs/adr/0024-the-component-library-is-shadcn-vue.md) ссылался на использование CLI `shadcn-vue`. Файл `modules/libs/ui/components.json` содержит мёртвые пути к несуществующим папкам. Фактически компоненты `@numen/ui` строятся напрямую на примитивах `reka-ui` и токенах Tailwind.
* **Решение:**
  - Переписать ADR-0024: зафиксировать прямое использование `reka-ui` примитивов и токенов Tailwind без внешнего CLI.
  - Удалить неработающий конфиг `modules/libs/ui/components.json`.

### 2.3. ADR-0015 и ADR-0016: Разделение понятий Book (EPUB) и Document (PDF)
* **Проблема:** В ADR-0015 и ADR-0016 термин `book` использовался как синоним для PDF-файлов и сканов с растрированием через `pdfium` и OCR. По глоссарию и ADR-0026 `book` — это исключительно EPUB (перетекающий текст в DOM, развороты, без OCR), а PDF — это `document`.
* **Решение:**
  - Актуализировать формулировки в ADR-0015 и ADR-0016, заменив смешанный термин `book` на `document` там, где речь идёт о растрировании и OCR.
  - Зафиксировать спецификацию EPUB Reflow Reader (клиентский рефлоу, колонки, отсутствие бэкенд-растрирования).

### 2.4. ADR-0020: Мультипроцессная работа с базой
* **Проблема:** ADR-0020 заявлял концепцию строго одного процесса (*«One process, one lifetime»*). ADR-0027 и ADR-0033 ввели два параллельных десктопных приложения (`numen` и `numen-flashcards`), разделяющих одну базу данных SQLite.
* **Решение:**
  - Добавить в ADR-0020 плашку `Amends:` со ссылками на ADR-0027 и ADR-0033, документирующую работу двух параллельных процессов с немедленными транзакциями SQLite.

### 2.5. ADR-0010: Кэширование текста для URL-источников
* **Проблема:** ADR-0010 утверждал: *«Ничто не хранит текст, отображение отрывка перечитывает исходный файл»*. Для источников типа `.url` (ADR-0038/0039) файл содержит только адрес, а текст извлекается из сети и кэшируется в служебной папке.
* **Решение:**
  - Добавить примечание в ADR-0010 о специфике URL-источников согласно ADR-0038/0039.

### 2.6. ADR-0006: Таксономия источников
* **Проблема:** В схеме `sources.kind` не упомянуты типы `document` и `url`.
* **Решение:**
  - Обновить ER-диаграмму и перечисление типов в ADR-0006 до всех 5 канонических источников: `note`, `book`, `document`, `recording`, `url`.

---

## 3. Удаление глоссария и синхронизация правил агентов

### 3.1. Удаление `docs/glossary.md`
- Файл `docs/glossary.md` удалён полностью. Все ссылки на него из ADR и `AGENTS.md` устранены.
- Ключевые предметные правила и различия зафиксированы напрямую в `AGENTS.md`.

### 3.2. Синхронизация `AGENTS.md`
- Синхронизирован единый корневой `AGENTS.md`:
  - Удалено устаревшее ограничение на ~200 строк файла.
  - Удалена устаревшая таблица запрещённых слов, дублировавшая старые ошибки глоссария.
  - Закреплены канонические правила именования, структура компонентов и слои.
- В `modules/libs/ui/package.json` обновлено описание на каноническое без устаревших ограничений.

---

## 4. Удаление мусора и мёртвых файлов

1. **`modules/apps/desktop/ui/`** — пустая директория-зомби (содержит только `.tsbuildinfo`). Настоящий редактор находится в `modules/apps/desktop/editor/`. **Удалить.**
2. **`modules/libs/ui/components.json`** — мёртвый конфиг shadcn-vue. **Удалить.**
3. **`docs/architecture/plan.md`** — устаревший черновик плана ветки `arch/layers-hold`. **Удалить.**

---

## 5. Нормализация нейминга в Go (хвосты 3-го лица)

В соответствии с правилом `AGENTS.md` Rule 2 (императивные глаголы вместо нарратива):

### 5.1. Экспортируемые методы домена:
- `Preset.Evens()` $\rightarrow$ `Preset.CanEvenLoad()`
- `Preset.Lands(...)` $\rightarrow$ `Preset.PlaceDueDate(...)`
- `Preset.Week()` $\rightarrow$ `Preset.GetWeeklyShare()`
- `Day.opens()` $\rightarrow$ `Day.getStartTime()`
- `Simulation.ripens / settles / reaches / Covers` $\rightarrow$ `getRipeningDays / settleDay / canReachGoal / GetDurationDays`
- `Config.KeepsCopiesInVault()` $\rightarrow$ `HasCopiesInVault()`
- `Appearance.Hangs()` $\rightarrow$ `HasPartsUnderNode()`
- `Indexing.Transcribes()` $\rightarrow$ `CanTranscribe()`
- `markdown.Differs()` $\rightarrow$ `Diff()`

### 5.2. Внутренние вспомогательные функции:
Заменить нарративные глаголы (`holds`, `takes`, `covers`, `stands`, `keeps`, `knows`, `spends`, `divides`, `deals`, `refuses`, `places`, `runs`, `says`) на конкретные инженерные глаголы (`recordShare`, `containsCard`, `ensureExists`, `applySettings`, `deductCost`, `distributeAllowance` и др.).

---

## 6. Порядок выполнения

- [ ] **Этап 1:** Удаление мёртвых файлов (`desktop/ui/`, `components.json`, `docs/architecture/plan.md`).
- [ ] **Этап 2:** Удаление `docs/glossary.md` и синхронизация `AGENTS.md`.
- [ ] **Этап 3:** Обновление и нормализация ADR (0044, 0024, 0015/0016, 0020, 0010, 0006).
- [ ] **Этап 4:** Рефакторинг Go-методов (3-е лицо $\rightarrow$ императив) с обновлением мест вызовов и тестов.
- [ ] **Этап 5:** Полный прогон ворот (`make lint`, `make test`, `depgraph`, `buf lint`, `vue-tsc`) и отправка коммита в PR #270.
