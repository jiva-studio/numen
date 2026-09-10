# Аудит кодовой базы и реестр проблем рефакторинга

Документ фиксирует текущий статус кодовой базы проекта Numen на основе правил `AGENTS.md`, `vue3-refactoring-review.md`, `refactoring-plan.md` и результатов глубокого аудита модулей.

---

## 1. Что уже выполнено (Fixed & Verified)

1. **Терминология внешних конфликтов файлов**:
   - `overtaken` унифицирован на `stale` во всех вкладках и хранилище заметок.
2. **Типизация ошибок и Result pattern (Frontend)**:
   - Создан модуль `shared/result.ts` с типом `Result<T, E = ErrorCode>`, конструкторами `ok()` и `err()`.
   - В `shared/note.ts`, `file.ts`, `vaults.ts` введены канонические типы `ErrorCode` и `VaultErrorCode`.
   - `refusalIn` переведен на `errorIn` в `shared/answers.ts` и `window/vault/`.
   - Добавлена обратная совместимость (псевдонимы `refusal`, `RefusalReason`).
3. **Именование реактивных фабрик и composables**:
   - `booking` $\rightarrow$ `useBookTab`
   - `openBook` $\rightarrow$ `useBookReader`
   - `documenting` $\rightarrow$ `useDocumentTab`
   - `openDocument` $\rightarrow$ `useDocumentReader`
   - `talking` $\rightarrow$ `useAgentConversation`
   - `filing` $\rightarrow$ `useFilesTab`
   - `decking` $\rightarrow$ `useDeckTabs`
4. **Стандартизация всех 26 Vue SFC компонентов**:
   - Все 26 компонентов приведены к структуре 4 секций:
     1. `// --- Props & Emits ---`
     2. `// --- State ---`
     3. `// --- Handlers ---` (все обработчики с префиксом `on<Action>`)
     4. `// --- Helpers ---`
   - Устранены анонимные inline-обработчики в шаблонах.
5. **Декомпозиция крупных модулей**:
   - `flashcards-deck-tab/deck.ts` (сокращён с 505 до 26 строк):
     - `types.ts` — типы `BufferDeck`, `BufferCard`, `BufferSection`, `NO_DECK`.
     - `serialize.ts` — чистая сериализация/десериализация (`deckOf`, `deckBodyOf`, `cardsOf`, etc.).
     - `drawn.ts` — маппинги UI-моделей и сравнения (`drawnOf`, `sameDeck`, etc.).
     - `mutations.ts` — чистые мутации карточек и секций (`added`, `dropped`, `filled`, etc.).
   - `flashcards-preset-tab/core.ts` (сокращён с 514 до 204 строк):
     - `types.ts` — все доменные типы, интерфейсы и константы (`Settings`, `Bounds`, `Preset`, `Curve`).
     - `core.ts` — только ConnectRPC-клиент и адаптер.
   - `document-tab/open.ts` (декомпозирован на `navigation.ts`, `viewport.ts`, `highlights.ts`, `types.ts`).
   - `book-tab/open.ts` (выделены типы в `types.ts`, пагинация в `pagination.ts`).
6. **Унификация ссылок**:
   - В `NoteResult` и `openNotes` введено каноническое свойство `link` (`LinkAddress`).
   - Удалён неиспользуемый дублирующий клиент `readings` в `document-tab/wire.ts`.

---

## 2. Актуальные проблемы в кодовой базе (Remaining Issues)

### Проблема A: Файлы с нарушением Single Responsibility (>200–400 строк)

Согласно правилу `AGENTS.md` (Rule 11), модуль не должен превышать ~200 строк и решать не связанные задачи. В проекте остаются файлы-монолиты:

1. **`modules/apps/desktop/editor/src/window/window.ts` (398 строк)**
   - «Бог-объект» окна: смешивает управление сеткой/раскладкой, историю фокуса вкладок, горячие клавиши, очередь сохранения при выходе, показ системных уведомлений и палитру команд.
   - *Решение*: разбить на `useWindowLayout`, `useWindowFocus`, `useWindowSaving`, `useWindowNotices`.
2. **`modules/apps/desktop/editor/src/note-tab/tab.ts` (438 строк)**
   - Смешивает состояние открытой вкладки заметки, интеграцию с CodeMirror/протоколом, отслеживание курсора, таймеры debounce и валидацию frontmatter.
   - *Решение*: разделить на `tabTypes.ts`, `useNoteEditor`, `useNoteFrontmatter`.
3. **`modules/apps/desktop/editor/src/note-tab/notes.ts` (378 строк)**
   - Хранилище всех заметок окна: держит карту тел, таймеры автосохранения, обработку внешних конфликтов на диске.
   - *Решение*: выделить очередь сохранения в `notesQueue.ts`, конфликты в `notesConflicts.ts`.
4. **`modules/apps/desktop/editor/src/shared/flashcards/cards.ts` (432 строки)**
   - Синтаксический анализ разметки карточек, сериализация в markdown, извлечение меток, валидация полей и клиент хранилища.
   - *Решение*: разделить на `cardParser.ts`, `cardSerializer.ts`, `cardsClient.ts`.
5. **`modules/apps/desktop/editor/src/shared/command/search.ts` (429 строк) и `palette.ts` (400 строк)**
   - Вся логика ранжирования поиска, фильтрации, палитры команд и подсказок собрана в двух огромных файлах.
   - *Решение*: декомпозировать алгоритмы скоринга и провайдеры команд.
6. **`modules/apps/desktop/editor/src/flashcards-stencil-tab/stencilTabs.ts` (359 строк)**
   - Монолитное состояние трафаретов карточек.
7. **`modules/apps/desktop/editor/src/flashcards-deck-tab/deckTabs.ts` (320 строк)**
   - Управление вкладками колод, привязка пресетов и реакция на таймеры.
8. **`modules/apps/desktop/editor/src/shared/words.ts` (313 строк)**
   - Гигантский монолитный словарь UI-строк: в одном файле свалены строки меню, редактора, настроек, ошибок, медиа-плеера и диалогов.
   - *Решение*: разбить словарь по доменам (`shared/words/editor.ts`, `shared/words/settings.ts`, etc.).
9. **`modules/apps/desktop/editor/src/window/vault/words.ts` (269 строк)**
   - Конвертер 12 различных protobuf-сообщений в UI-модели в одном файле.

---

### Проблема B: Литературные и глагольные имена в доменных моделях (Rule 2)

Правило `AGENTS.md` (Rule 2) запрещает:
- Причастия прошедшего времени (`spoken`, `listened`, `answered`, `written`, `seen`)
- Формы 3-го лица (`carries`, `attends`, `reads`, `opens`)
- Литературные метафоры (`minting`, `held`, `cold`)

Где это ещё встречается:
1. **`window/vault/words.ts`**:
   - Методы: `answered`, `around`, `filed`, `written`, `seenOf`.
   - *Исправление*: `mapNoteResult`, `mapNeighbourhood`, `mapFileEntry`, etc.
2. **`note-tab/tab.ts` и `notes.ts`**:
   - Свойства и методы: `said`, `told`, `over`, `attends`, `shown`, `typed`, `carries`.
   - *Исправление*: `content`, `lastNotified`, `target`, `focus`, `isShown`, `typeText`.
3. **`shared/command/`**:
   - `wants`, `stands`, `said`.
   - *Исправление*: `requires`, `location`, `message`.

---

### Проблема C: Дублирование терминологии адресов и ссылок (Rule 9)

- В коде всё ещё параллельно сосуществуют термины `address` и `link`:
  - `shared/command/address.ts` $\rightarrow$ переименовать в `url.ts` / `link.ts`.
  - `isWebAddress()` $\rightarrow$ `isWebUrl()` / `isLink()`.
  - `importAddress()` в `files-tab/types.ts` $\rightarrow$ `importUrl()`.
  - `carriesAddress()` в `files-tab/drag.ts` $\rightarrow$ `carriesUrl()`.
  - Свойства `address` в типах заметок и истории $\rightarrow$ окончательно мигрировать на `link`.

---

### Проблема D: Protobuf и Go Core (Сквозная нормализация)

1. **Protobuf (`modules/libs/protocol/proto/numen/v1/`)**:
   - `enum Refusal` $\rightarrow$ переименовать в `enum ErrorCode`.
   - `enum VaultsRefusal` $\rightarrow$ `enum VaultsErrorCode`.
   - Поля ответов: `Refusal refusal = ...` $\rightarrow$ `ErrorCode error = ...`.
   - Требуется кодогенерация через `buf generate`.
2. **Go Core (`modules/libs/core/`)**:
   - В интерфейсах портов (`port/notes.go`, `port/vaults.go`) возвращаются внутренние структуры отказов (`refusal`) вместо стандартного Go-интерфейса `error`.
   - Методы адаптеров в ряде мест используют метафорические имена вместо стандартных глаголов (`Get`, `Read`, `Write`, `Create`).

---

### Проблема E: Тестовые файлы-гиганты (>500–1400 строк)

Тесты содержат дублирующиеся фикстуры и разрослись до огромных размеров:
- `modules/apps/desktop/editor/src/plex-tab/kind.test.ts` — **1409 строк**
- `modules/apps/desktop/editor/src/flashcards-preset-tab/kind.test.ts` — **864 строки**
- `modules/apps/desktop/editor/src/files-tab/kind.test.ts` — **633 строки**
- `modules/apps/desktop/editor/src/note-tab/kind.test.ts` — **535 строк**
- `modules/apps/desktop/editor/src/shared/testing/window.ts` — **669 строк** (общий мок окна)

---

## 3. Рекомендуемый план на следующий раунд рефакторинга

1. **Этап 1: Декомпозиция оставшихся крупных модулей ядра десктопа**:
   - `window/window.ts` (398 строк) $\rightarrow$ разделение на composables фокуса, раскладки и сохранения.
   - `note-tab/tab.ts` (438 строк) и `notes.ts` (378 строк) $\rightarrow$ вынос типов и очереди сохранения.
   - `shared/flashcards/cards.ts` (432 строки) $\rightarrow$ парсер разметки и сериализатор.
2. **Этап 2: Чистка литеральной терминологии в доменных объектах**:
   - Замена причастий прошедшего времени (`answered`, `written`, `said`, `told`, `typed`) на понятные инженерные предикаты и методы.
3. **Этап 3: Полная унификация `address` $\rightarrow$ `link` / `url`**:
   - В `shared/command/`, `files-tab/drag.ts`, `files-tab/types.ts`.
4. **Этап 4: Разделение словаря локализации `shared/words.ts` (313 строк)**:
   - Модуляризация словаря по доменным подсистемам.
