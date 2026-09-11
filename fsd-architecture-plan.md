# Numen Desktop Editor — Комплексный план рефакторинга (PR #270)

Ветка: `feat/a-book-made-for-a-screen-is-read`  
Рабочая директория: `modules/apps/desktop/editor/src/`  
Основано на: `review3.md`, `refactor2.md`, `AGENTS.md`

---

## 1. Фундаментальные правила и запреты

1. **Никакой обратной совместимости (Zero Compatibility Shims)**:
   - Полный запрет на псевдонимы `trouble = error`, `refusal = error`, дублирующие типы `PageHighlight = HighlightPage`.
   - Рефакторятся сразу все потребители и тесты.
2. **Именование (AGENTS.md п. 2, 3, 5, 12)**:
   - Каталоги: `kebab-case`.
   - Vue-компоненты: `PascalCase`.
   - TypeScript-файлы: `camelCase`, composables: строго `use<Feature>.ts`.
   - Функции: инфинитив глагола в базовой форме (`add`, `remove`, `get`, `create`, `update`). Никаких 3-го лица (`adds`, `steers`) и герундиев (`adding`, `listening`).
   - Ивенты: `on<Action>` (`onSelect`, `onSubmit`, `onClose`).
   - Булевы поля: `is*`, `has*`, `can*` (`isRenaming`, `hasTranscript`).
3. **Размер компонентов**:
   - Каждый `.vue` файл $\le 250$ строк.
4. **Структура `<script setup>`**:
   - Четыре секции: `Props & Emits` $\rightarrow$ `State` $\rightarrow$ `Handlers` $\rightarrow$ `Helpers`.

---

## 2. Ликвидация псевдопоэтических метафор и архаичных типов

| В коде сейчас | Что это в реальности | Новое чистое имя |
|---|---|---|
| `interface Went { from, to }` | Запись о переименовании/перемещении файла | **`PathRename`** |
| `wentTo(renamed, path)` | Функция вычисления нового пути | **`getRenamedPath(renamed, path)`** |
| `interface Made { path, error }` | Результат создания сущности | **`CreateResult`** |
| `interface Said { text, from, to }` | Потоковый патч/дифф текста | **`TextPatch`** |
| `tickets` (`plex-tab/`) | Маппинг путей на стабильные строковые ID узлов | **`nodeIdMap`** / **`usePlexNodeIds`** |
| `standing` (`plex-tab/`) | Активный центральный узел графа и окружение | **`focus`** / **`usePlexFocus`** |
| `mint / minting` | Генерация ID сущности или ноды | **`generateId`** / **`createId`** |
| `stretch / stretches` | Диапазон/срез текста | **`span`** (`Span { from, to }`) |
| `overtaken` | Параллельное внешнее изменение файла на диске | **`stale`** / **`conflict`** |
| `trouble` | Текст ошибки операции | **`error`** / **`errorMessage`** |
| `putting` | Маршрутизатор открытия файлов по типам | **`useFileRouter`** |
| `talk` (`agent-tab/`) | Сессия диалога с ассистентом | **`conversation`** / **`chat`** |

---

## 3. Интеграция 44 пунктов из `review3.md` по модулям

### 3.1. `shared/` и ядро
- [ ] `shared/core.ts:27`: Замена `Went` $\rightarrow$ `PathRename`, `wentTo` $\rightarrow$ `getRenamedPath`, `Made` $\rightarrow$ `CreateResult`, `Said` $\rightarrow$ `TextPatch`.
- [ ] `shared/command/handlers.ts:19`: Устранить наречия и герундии в названиях хэндлеров.
- [ ] Полное искоренение `trouble` и `refusal` из `shared/` без сохранения алиасов.

### 3.2. `document-tab/` (PDF-читалка)
- [ ] `document-tab/types.ts:21, 27, 35`: Удалить дублирующиеся типы (`PageHighlight`, `HighlightPage`), **удалить все типы обратной совместимости**.
- [ ] `document-tab/highlights.ts:12, 13, 29, 37, 53, 55, 57`:
  - `at` $\rightarrow$ `pageNumber`
  - `trouble` $\rightarrow$ `error`
  - `reach` $\rightarrow$ нормальный глагол
  - Убрать `also`, нормальные имена для подсветки.
- [ ] `document-tab/navigation.ts`: Переименовать файл в `useDocumentNavigation.ts`:
  - `at` $\rightarrow$ `pageNumber`
  - `go` $\rightarrow$ `goToPage`
  - `next` $\rightarrow$ `nextPage`, `back` $\rightarrow$ `prevPage`.
- [ ] `document-tab/open.ts:36`: Переименовать в `useDocumentTab.ts`, декомпозировать composable.
- [ ] `document-tab/viewport.ts:13, 14, 29, 36, 37`: Убрать `seen`, понятные имена для URL страниц.

### 3.3. `book-tab/` (EPUB-читалка)
- [ ] `book-tab/BookTab.vue:74, 96, 113, 115`:
  - Вынести содержимое чтения из transition в отдельный дочерний компонент **`BookContent.vue`**.
  - `at` $\rightarrow$ `currentPage`.
  - Убрать `elseware` $\rightarrow$ объединить/переименовать.
  - Поднять хуки вверх по стандарту структуры.
- [ ] `book-tab/markup.ts:17`: `pointed at` $\rightarrow$ функция с глаголом `get*`.
- [ ] `book-tab/open.ts:32, 51, 52, 54`:
  - Убрать лишний тип `bookspan` в пользу единого `Span`.
  - `at` $\rightarrow$ `currentPage`, убрать `seen`, `standing`.
- [ ] `book-tab/types.ts:50`: `readnMarkup` $\rightarrow$ `getMarkup`, убрать `seen`.

### 3.4. `files-tab/` (Файловый менеджер)
- [ ] `files-tab/FilesTab.vue:34, 142`:
  - `renaming` $\rightarrow$ `isRenaming` (булево).
  - `trouble` $\rightarrow$ `error`.
- [ ] `files-tab/listing.ts:27, 42, 46, 55, 66, 74`:
  - `landed in` $\rightarrow$ понятное имя функции.
  - Названия функций — глаголы без наречий и герундиев.
  - Переименовать файл и composable в `useFileList.ts`.
  - `trouble` $\rightarrow$ `error`.
- [ ] `files-tab/open.ts:28, 166`:
  - Декомпозировать большой composable, убрать алиасы экспортов.
  - Все event-handlers назвать `on<Action>`.
- [ ] `files-tab/types.ts:14, 18, 21, 22, 32, 40, 65`:
  - Убрать `intoybefore`, разбить большой тип на независимые интерфейсы.
  - Убрать `lands`, `seis`, `asking` $\rightarrow$ глаголы.

### 3.5. `flashcards-deck-tab/` (Повторение колоды)
- [ ] `flashcards-deck-tab/DeckTab.vue:23, 29, 123, 129`:
  - `trouble` $\rightarrow$ `error`.
  - `offered` $\rightarrow$ нормальное имя поля.
  - Вынести строки списка карточек в отдельный компонент **`DeckCardRow.vue`** (уменьшить компонент $\le 250$ строк).
- [ ] `flashcards-deck-tab/answers.ts:125`: Функции назвать глаголами без герундиев.
- [ ] `flashcards-deck-tab/deckTabs.actions.ts:19, 25, 26, 45`:
  - Переименовать в `useDeckActions.ts`.
  - Устранить дубликаты (`removeSection` vs `removesSection`).
  - Функции в базовой форме глагола (`add`, не `adds`).
- [ ] `flashcards-deck-tab/deckTabs.schedule.ts:12, 57`:
  - Переименовать в `useDeckSchedule.ts`, нормальные глаголы.
- [ ] `flashcards-deck-tab/deckTabs.ts:34, 180, 260`:
  - Переименовать в `useDeckTabs.ts`, разбить.
  - `minting` $\rightarrow$ `generateId`.
  - Понятные имена возвращаемых переменных.
- [ ] `flashcards-deck-tab/mutations.ts:25`:
  - Разбить на сфокусированные модули, `mint` $\rightarrow$ `generateId`.
- [ ] `flashcards-deck-tab/reader.ts:14, 74`: Понятное имя типа, функция `set*`.
- [ ] `flashcards-deck-tab/types.ts:69`: Разбить мега-тип на несколько стейтов.

### 3.6. `flashcards-preset-tab/` (Настройка алгоритма SRS)
- [ ] `flashcards-preset-tab/PresetTab.vue:31, 37, 81, 85, 99, 107, 118`:
  - Убрать `said instead`.
  - Прямой доступ к реактивным полям без костыльных конвертеров.
  - Вынести секции в дочерние компоненты (компонент строго $\le 250$ строк).
- [ ] `flashcards-preset-tab/core.ts:17`: `refusal` $\rightarrow$ `error`.
- [ ] `flashcards-preset-tab/kind.ts:97`: Переименовать в `usePresetTab.ts`.

### 3.7. `flashcards-stencil-tab/` (Шаблоны карточек)
- [ ] `flashcards-stencil-tab/StencilTab.vue:89, 126`: Декомпозировать шаблон на подкомпоненты.

### 3.8. `note-tab/` (Заметки)
- [ ] `note-tab/kind.ts:166, 182`: Переименовать в `useNoteTab.ts`, убрать наречия.
- [ ] `note-tab/notes.ts:155`: Искоренить герундии, глаголы в базовой форме.

### 3.9. `plex-tab/` (Граф связей)
- [ ] `plex-tab/PlexTab.vue:128, 129`: Вынести подкомпоненты, декомпозировать.
- [ ] `plex-tab/kind.ts:111`:
  - `standing` $\rightarrow$ `focus` / `usePlexFocus`.
  - `tickets` $\rightarrow$ `nodeIds` / `usePlexNodeIds`.

### 3.10. `settings-file-tab/` $\rightarrow$ `text-editor/`
- [ ] `settings-file-tab/SettingsFileTab.vue:44`:
  - Рефакторинг в универсальный текстовый редактор **`TextEditorTab.vue`** (`useTextEditor.ts`).
  - Поддержка параметров: путь и `language` (JSON, Markdown, YAML).

### 3.11. `settings-tab/` (Визуальные настройки)
- [ ] `settings-tab/kind.ts:17`: Переименовать в `useSettingsTab.ts`.
- [ ] `settings-tab/setting-row/SettingRow.vue:16`: Проверить именование пропсов и эмиттеров (`on<Action>`).

### 3.12. `agent-tab/` (AI-чат)
- [ ] `agent-tab/kind.ts:22, 24, 38, 47, 48, 51`:
  - `talk` $\rightarrow$ `conversation`.
  - `asked` $\rightarrow$ `userInput` / `query`.
  - `place` $\rightarrow$ `channel` / `session`.
  - `landed`, `asking` $\rightarrow$ нормальные типы и имена.
  - Устранить дублирование с `places.ts` и `types.ts`.

### 3.13. `App.vue` (Корневой шелл)
- [ ] `modules/apps/desktop/editor/src/App.vue:47`:
  - Декомпозиция монолита: вынести инициализацию в `useAppBootstrap`, шорткаты в `useAppHotkeys`.
  - Переименовать функции: убрать `doing` $\rightarrow$ `tasks`, `going` $\rightarrow$ `navigate`, искоренить герундии.
  - Привести размер `App.vue` к $\le 200$ строк.

---

## 4. Порядок поэтапного выполнения

1. **Фаза 1: Shared Core & Terminology** (`core.ts`, замена `Went`, `Made`, `Said`, `trouble` $\rightarrow$ `error`).
2. **Фаза 2: `document-tab/` & `book-tab/`** (удаление обратной совместимости, `BookContent.vue`, `useDocumentNavigation.ts`).
3. **Фаза 3: `files-tab/` & `settings-file-tab/`** (`isRenaming`, `useFileList.ts`, превращение в универсальный `TextEditorTab.vue`).
4. **Фаза 4: `flashcards-deck-tab/`, `flashcards-preset-tab/`, `flashcards-stencil-tab/`** (декомпозиция компонентов $\le 250$ строк, вынос `DeckCardRow.vue`, чистка `mutations.ts`).
5. **Фаза 5: `note-tab/`, `plex-tab/`, `agent-tab/`, `settings-tab/`** (`focus`, `nodeIds`, `conversation`).
6. **Фаза 6: `App.vue`** (декомпозиция в тонкий шелл).
7. **Фаза 7: Полная верификация** (`npm run test:unit`, `npm run build`, `npm run test:stories`).
