# Numen Desktop Editor — Реестр архитектурных долгов и замечаний Review3

> Файл сформирован на основе детального аудита кодовой базы против `review3.md`, `fsd-architecture-plan.md` и `AGENTS.md`.  
> Ветка: `feat/a-book-made-for-a-screen-is-read`  
> Рабочая директория: `modules/apps/desktop/editor/src/`

---

## 1. Вынос типов и разделение моделей (Type Extraction)

Замечание ревью: *«Типы по файлам не вынес. Опять набор типов в одном файле, раскидай по файлам»*.

- [x] **`shared/core.ts`**:
  - [x] Разбить гигантский интерфейс `Core` (God Object) на узкие роли/порты по доменам (`NotePort`, `VaultPort`, `SettingsPort`, `FilePort`).
  - [x] Вынести связанные типы (`NoteResult`, `Neighbourhood`, `Task`, `Configuration`) в соответствующие доменные сущности `entities/`.
- [x] **`shared/note.ts`**:
  - [x] Выделить типы (`NoteResult`, `NewNote`, `NoteHeading`, `Link`, `NoteEdit`, `Span`) в `entities/note/types.ts`.
  - [x] Отделить чистые структуры данных от функций мутаций (`findHeadings`, `replaceBody`).
- [x] **`shared/artifacts.ts`**:
  - [x] Вынести типы `Artifact`, `ArtifactState`, `ArtifactStates`, `Outcome` в `shared/types/artifacts.ts`.
  - [x] Оставить в `artifacts.ts` только клиенты и методы запуска.
- [x] **`widgets/file-manager`**:
  - [x] Вынести типы `Folders`, `ListingRow` из `listing.ts` в `types.ts`.
  - [x] Устранить тип `intoybefore` в `types.ts`, разбить монолитные типы стейтов на узкие независимые интерфейсы.
- [x] **`widgets/deck-editor`**:
  - [x] Вынести стейт колоды и параметры действий из `deckTabs.ts` в `types.ts`.
  - [x] Разбить мега-тип `DeckTabState` (сейчас более 20 полей) на независимые стейты.
- [x] **`widgets/stencil-editor`**:
  - [x] Разбить мега-стейт `StencilTabState` (30+ полей) на узкие подтипы.
- [x] **`widgets/preset-editor`**:
  - [x] Разбить монолитный `types.ts` (более 200 строк с дублирующимися типами для чтения/записи) на сфокусированные типы.
- [x] **`widgets/plex-graph`**:
  - [x] Разбить `types.ts` (со стейтом на 50+ полей и кучей зависимостей) на отдельные типы в `entities/plex/` или внутри виджета.
- [x] **`widgets/book-reader`**:
  - [x] Вычистить тип `BookSpan` в пользу единого системного `Span` (`{ from, to }`).
  - [x] Убрать `printedPages`, `pageBytes`, привести типы книги к канонической модели EPUB.

---

## 2. Исправление корявых и архаичных неймингов (Nomenclature Standardization)

Замечание: *«Нейминги такие же корявые, опять глаголы в 3-м лице, герундии и псевдопоэтические метафоры»*.

### 2.1. Методы в 3-м лице и герундии в `Core` (`shared/core.ts`)
Правило `AGENTS.md` п. 2 запрещает формы 3-го лица (`chooses*`, `writes*`) и герундии.
- [x] `core.choosesSyncing` $\rightarrow$ `setSyncEnabled` (или `updateSyncSetting`)
- [x] `core.syncing` $\rightarrow$ `getSyncEnabled`
- [x] `core.choosesHanging` $\rightarrow$ `setHangingSettings`
- [x] `core.hanging` $\rightarrow$ `getHangingSettings`
- [x] `core.choosesReviewing` $\rightarrow$ `setReviewSettings`
- [x] `core.reviewing` $\rightarrow$ `getReviewSettings`
- [x] `core.choosesSetting` $\rightarrow$ `updateSettings`
- [x] `core.writesSettingsFile` $\rightarrow$ `saveSettingsFile`
- [x] `core.settingsFile` $\rightarrow$ `getSettingsFile`
- [x] `core.attending` $\rightarrow$ `setFocus`
- [x] `core.opening` $\rightarrow$ `getInitialOpenPath`

### 2.2. Метафоры и корявые имена в `plex-graph`
- [x] **`tickets.ts`** $\rightarrow$ переименовать файл в `nodeIdMap.ts` (или composable `usePlexNodeIds.ts`).
  - [x] `tickets.of(path)` $\rightarrow$ `getNodeId(path)`
  - [x] `tickets.note(id)` $\rightarrow$ `getNodePath(id)`
  - [x] `tickets.keeps(...)` $\rightarrow$ `retainNodeIds(...)`
  - [x] `tickets.moved(...)` $\rightarrow$ `updateRenamedNodes(...)`
- [x] **`standing`** $\rightarrow$ заменить на `focus` / `activeNode`:
  - [x] Заменить во всех обращениях и вызовах в `PlexTab.vue` и хуках.
- [x] **`PlexTab.vue`**:
  - [x] `dropName` $\rightarrow$ переименовать в `targetNodeId` / `targetPath`.
  - [x] Передавать части/порты внутри типизированного узла `PlexNode`, а не отдельным параллельным массивом.

### 2.3. Корявые имена в `document-viewer` (PDF)
- [x] **`navigation.ts`**:
  - [x] `at` $\rightarrow$ `pageNumber`
  - [x] `go` $\rightarrow$ `goToPage`
  - [x] `next` $\rightarrow$ `nextPage`
  - [x] `back` $\rightarrow$ `prevPage`
- [x] **`highlights.ts`**:
  - [x] `at` $\rightarrow$ `pageNumber`
  - [x] Функция `highlight` $\rightarrow$ переименовать в императивный глагол `renderHighlight` / `applyHighlight`.
  - [x] Убрать `also` $\rightarrow$ сделать типизированные виды подсветки (`HighlightKind`).

### 2.4. Корявые имена в `book-reader` (EPUB)
- [x] **`BookTab.vue` & `open.ts`**:
  - [x] `at` $\rightarrow$ `currentPage`
  - [x] Убрать `elseware` $\rightarrow$ объединить с подсветкой или дать понятное имя (`searchHighlights`).
  - [x] `seen` $\rightarrow$ убрать архаизм, использовать понятный `loadedContent`.
  - [x] `standing` $\rightarrow$ `currentChapter` / `activeSection`.
- [x] **`markup.ts`**:
  - [x] `pointed at` $\rightarrow$ функция `getLinkTarget` / `resolveLink`.
- [x] **`pagination.ts`**:
  - [x] `named` $\rightarrow$ глагол `getPageTitle` / `formatPageName`.

### 2.5. Корявые имена в `file-manager`
- [x] **`listing.ts`**:
  - [x] `landedIn` $\rightarrow$ `resolveDropFolder`
  - [x] `above` $\rightarrow$ `getParentFolders`
  - [x] `folderOf` $\rightarrow$ `getFolderPath`
  - [x] `freeName` $\rightarrow$ `generateUniqueName`
- [x] **`open.ts` & `types.ts`**:
  - [x] Убрать экспорт через псевдонимы (алиасы).
  - [x] `lands`, `seis`, `asking` $\rightarrow$ глаголы в инфинитиве (`resolveDrop`, `isAvailable`, `confirm`).
  - [x] `renaming` $\rightarrow$ `isRenaming`.

### 2.6. Корявые имена и дубликаты в `deck-editor` и `stencil-editor`
- [x] **`deckTabs.ts`**:
  - [x] `minting` $\rightarrow$ `generateId` / `createCardId`.
- [x] **`deckTabs.actions.ts`**:
  - [x] Устранить глаголы 3-го лица: `adds` $\rightarrow$ `add`, `removesSection` $\rightarrow$ дубликат `removeSection`.
  - [x] Устранить дублирование логики между экшенами.
- [x] **`stencilTabs.ts`**:
  - [x] Устранить дубликаты методов: `addFace` / `addsFace`, `moveField` / `movesField`.
  - [x] Именовать хэндлеры `on<Action>`.

### 2.7. Корявые имена в `App.vue` и корневом шелле
- [x] `doing` $\rightarrow$ `tasks`
- [x] `going` $\rightarrow$ `navigate`
- [x] `where` $\rightarrow$ `currentVault`
- [x] `carries` $\rightarrow$ `artifactStates`
- [x] `places` $\rightarrow$ `openTabs`
- [x] `shut` $\rightarrow$ `closeTab`

---

## 3. Переименование файлов Composables в `use<Feature>.ts`

Правила 5 и 12 `AGENTS.md`: *«Composables naming: use<Feature>, wrong: booking, documenting, talking, opening. Имя файла должно строго соответствовать имени composable»*.

- [x] `widgets/book-reader/kind.ts` $\rightarrow$ **`useBookTab.ts`** (+ `useBookTab.test.ts`)
- [x] `widgets/book-reader/open.ts` $\rightarrow$ **`useBookReader.ts`** (+ `useBookReader.test.ts`)
- [x] `widgets/document-viewer/kind.ts` $\rightarrow$ **`useDocumentTab.ts`** (+ `useDocumentTab.test.ts`)
- [x] `widgets/document-viewer/open.ts` $\rightarrow$ **`useDocumentReader.ts`** (+ `useDocumentReader.test.ts`)
- [x] `widgets/document-viewer/navigation.ts` $\rightarrow$ **`useDocumentNavigation.ts`** (+ `useDocumentNavigation.test.ts`)
- [x] `widgets/document-viewer/highlights.ts` $\rightarrow$ **`useDocumentHighlights.ts`** (+ `useDocumentHighlights.test.ts`)
- [x] `widgets/document-viewer/viewport.ts` $\rightarrow$ **`useDocumentViewport.ts`** (+ `useDocumentViewport.test.ts`)
- [x] `widgets/note-editor/kind.ts` $\rightarrow$ **`useNoteTab.ts`** (+ `useNoteTab.test.ts`)
- [x] `widgets/file-manager/open.ts` $\rightarrow$ **`useFilesTab.ts`** (+ `useFilesTab.test.ts`)
- [x] `widgets/file-manager/listing.ts` $\rightarrow$ **`useFileTree.ts`** (+ `useFileTree.test.ts`)
- [x] `widgets/deck-editor/deckTabs.ts` $\rightarrow$ **`useDeckTabs.ts`** (+ `useDeckTabs.test.ts`)
- [x] `widgets/deck-editor/scheduler.ts` $\rightarrow$ **`useDeckSchedule.ts`** (+ `useDeckSchedule.test.ts`)
- [x] `widgets/deck-editor/deckTabs.schedule.ts` $\rightarrow$ **`useDeckScheduleWiring.ts`**
- [x] `widgets/preset-editor/kind.ts` $\rightarrow$ **`usePresetTab.ts`** (+ `usePresetTab.test.ts`)
- [x] `widgets/stencil-editor/stencilTabs.ts` $\rightarrow$ **`useStencilTabs.ts`** (+ `useStencilTabs.test.ts`)
- [x] `widgets/plex-graph/open.ts` $\rightarrow$ **`usePlexTab.ts`** (+ `usePlexTab.test.ts`)
- [x] `widgets/plex-graph/view.ts` $\rightarrow$ **`usePlexView.ts`** (+ `usePlexView.test.ts`)
- [x] `widgets/plex-graph/parts.ts` $\rightarrow$ **`usePlexParts.ts`** (+ `usePlexParts.test.ts`)
- [x] `widgets/agent-chat/kind.ts` $\rightarrow$ **`useAgentConversation.ts`** (+ `useAgentConversation.test.ts`)
- [x] `widgets/text-editor/kind.ts` $\rightarrow$ **`useTextEditor.ts`** (+ `useTextEditor.test.ts`)
- [x] `app/window.ts` $\rightarrow$ **`useWindow.ts`**
- [x] `app/commands.ts` $\rightarrow$ **`useCommands.ts`**
- [x] `app/showing.ts` $\rightarrow$ **`useWindowShowing.ts`**
- [x] `app/kinds.ts` $\rightarrow$ **`useWindowKinds.ts`**
- [x] `app/streams.ts` $\rightarrow$ **`useWindowStreams.ts`**
- [x] `app/editing.ts` $\rightarrow$ **`useEditing.ts`**
- [x] `app/settings.ts` $\rightarrow$ **`useSettings.ts`**
- [x] `app/notices.ts` $\rightarrow$ **`useWindowNotices.ts`**
- [x] `app/attention.ts` $\rightarrow$ **`useAttention.ts`**
- [x] `app/vaults.ts` $\rightarrow$ **`useVaults.ts`**

---

## 4. Декомпозиция крупных Vue-компонентов и раздутых Composables

- [x] **`BookTab.vue`**:
  - [x] Выделить компонент содержимого чтения **`BookContent.vue`** (убрать нагромождение разметки из `transition`).
- [x] **`DeckTab.vue`**:
  - [x] Выделить строки списка карточек в отдельный дочерний компонент **`DeckCardRow.vue`**.
- [x] **`PresetSettings.vue`**:
  - [x] Разбить компонент на 300+ строк на сфокусированные секции настроек алгоритма.
- [x] **`App.vue`**:
  - [x] Вынести начальную инициализацию в `useAppBootstrap.ts`.
  - [x] Вынести горячие клавиши в `useAppHotkeys.ts`.
- [x] **Декомпозиция composables с огромным количеством `return`**:
  - [x] `note-editor/notes.ts`: разбить composable с 25+ возвращаемыми полями на узкие хуки (`useNoteContent`, `useNoteFrontmatter`, `useNoteAutoSave`).
  - [x] `plex-graph/open.ts`: разбить composable с 30+ полями на узкие хуки (`usePlexSelection`, `usePlexLayout`, `usePlexDrag`).
  - [x] `file-manager/open.ts`: разбить composable на `useFileSelection`, `useFileOperations`.

---

## 5. Физический перенос по слоям FSD (Очистка `shared/`)

Замечание: *«Все в shared уехало, entities и features пустые»*.

- [x] **`features/`**:
  - [x] Создать `features/command-palette/`: перенести туда всю палитру, поиск, фильтрацию команд и хэндлеры из `shared/command/`.
  - [x] Создать `features/file-conflict/`: перенести логику обнаружения устаревания (`stale`) и сброса на диск из `shared/saving/`.
- [x] **`entities/`**:
  - [x] `entities/note/`: модель заметки, парсинг заголовков, frontmatter (из `shared/note.ts`).
  - [x] `entities/deck/`: модель колоды, карточек, повторений, расчет интервалов (из `shared/flashcards/`).
  - [x] `entities/media/`: плеер, транскрипты, cues, тайминги (из `shared/media/`).
  - [x] `entities/settings/`: внешний вид, темы, синхронизация (из `shared/settings/`).
  - [x] `entities/tab/`: модели и состояние вкладок (из `shared/tabs/`).
- [x] **`shared/` (очистка)**:
  - В `shared/` должны остаться исключительно системная инфраструктура:
    - `shared/types/` (`core.ts`, `result.ts`, `errors.ts`)
    - `shared/lib/` (`paths.ts`, `words/`)
    - `shared/api/` (`artifacts.ts`, ConnectRPC transport)
    - `shared/ui/` (`icons.ts`, базовые уведомления)
