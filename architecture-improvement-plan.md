# Numen Desktop Editor — Комплексный план архитектурных улучшений
## Clean Architecture, Hexagonal Architecture, FSD, DDD и декомпозиция крупных Composables

**Дата составления:** 11 сентября 2026  
**Ветка:** `feat/a-book-made-for-a-screen-is-read`  
**Локация проекта:** `modules/apps/desktop/editor/src/`  
**Статус текущего набора тестов:** 2031 / 2031 passed (100% green), 33 / 33 lint tests passed (100% green).

---

## 1. Executive Summary & Текущее состояние

В ходе предыдущих этапов рефакторинга были успешно решены ключевые проблемы:
- Полностью устранены алиасы и шимы обратной совместимости (`export const A = B`).
- Исправлены нарушения терминологии предметной области:
  - `Problem` $\rightarrow$ `DeckProblem` (вместо абстрактного глобального имени).
  - `places.ts` $\rightarrow$ `shared/links.ts` (выделены чистые функции парсинга и сопоставления ссылок: `parseLinkTarget`, `extractLinkTargets`, `areLinkTargetsEqual`).
- Разделены сервисы `DeckService` и `StencilService`.
- Внедрена внутренняя файловая структура срезов с разделением на `components/` и `composables/` в первых виджетах (`widgets/deck-editor/`, `widgets/settings/`, `widgets/stencil-editor/`, `widgets/agent-chat/`).

Однако для достижения максимальной чистоты, модульности и поддерживаемости кода необходимо систематически решить проблемы крупноразмерных composables с избыточным количеством экспортируемых полей, устранить межвиджетную связанность и довести структуру до эталонного соответствия Clean Architecture, Hexagonal Architecture, FSD и DDD.

---

## 2. Анализ по архитектурным методологиям

### 2.1 Clean & Hexagonal Architecture (Гексагональная архитектура / Порты и Адаптеры)

| Концепция | Текущее состояние в проекте | Выявленные проблемы | Целевое решение |
|---|---|---|---|
| **Domain Entities (Ядро)** | `entities/deck`, `entities/note`, `entities/media`, `entities/settings`, `entities/tab` | Высокая чистота типов, но в некоторых модулях бизнес-логика перемешана с адаптерами Wire. | Полная изоляция сущностей от `@connectrpc` и транспортных протоколов. Сущности не импортируют транспорт. |
| **Ports (Интерфейсы)** | `DeckService`, `StencilService`, `ShownStore`, `FileOpeners`, `WindowHandle` | Интерфейсы портов частично объявлены рядом с конкретными адаптерами или внутри виджетов. | Вынесение портов в чистые интерфейсные файлы сущностей (`entities/<domain>/ports.ts` или `types.ts`). |
| **Adapters (Адаптеры)** | ConnectRPC клиенты (`@numen/wire`, `cardsService`), UI компоненты (`@numen/ui`) | В `entities/deck/cards.ts` клиент `cardsService` инстанциируется прямо в модуле, что затрудняет мокирование и нарушает Dependency Inversion. | Внедрение зависимостей (DI) через фабрики адаптеров: передача `client` или `transport` через параметры конфигурации. |
| **Use Cases (Сценарии)** | Оркестрация в `app/useCommands.ts`, `app/useWindowKinds.ts`, `app/useNoteEditors.ts` | Сценарии перегружены: один composable решает задачи и роутинга, и сохранения, и палитры команд. | Разделение use-case composables по единой ответственности (SRP). |

### 2.2 Feature-Sliced Design (FSD)

Текущая слоистая архитектура `src/`:
1. `app/` — Инициализация, общие провайдеры, связывание слоев.
2. `widgets/` — Композитные UI-блоки (вкладки приложения).
3. `features/` — Пользовательские сценарии (command-palette, file-conflict).
4. `entities/` — Бизнес-сущности и их сервисы.
5. `shared/` — Базовая инфраструктура, системные типы, утилиты.

#### Выявленные нарушения FSD:
1. **Межвиджетные импорты (Cross-Widget Dependencies)** — прямое нарушение FSD (слой не должен зависеть от своего же слоя):
   - `widgets/deck-editor` импортирует из `widgets/preset-editor/core` и `widgets/note-editor/notes`.
   - `widgets/stencil-editor` импортирует из `widgets/note-editor/notes`.
   - **Решение:** Общие концепции (например, `openNotes`, пресеты расписания) должны быть подняты в соответствующие `entities/note/` и `entities/deck/` либо абстрагированы через порты.
2. **Анатомия срезов (Slice Anatomy):**
   - Успешно внедрена схема `components/` + `composables/` в 4 виджетах.
   - Остальные виджеты (`book-reader`, `document-viewer`, `file-manager`, `note-editor`, `plex-graph`, `preset-editor`) пока имеют плоскую структуру файлов.
   - **Решение:** Провести аналогичную реорганизацию во всех оставшихся виджетах.

### 2.3 Domain-Driven Design (DDD)

#### Bounded Contexts (Ограниченные контексты):
1. **Note Context:** Управление Markdown-заметками, заголовками, сохранением (`entities/note`, `widgets/note-editor`).
2. **Flashcard Deck Context:** Карточки, трафареты (stencils), поля, валидация колод (`entities/deck`, `widgets/deck-editor`, `widgets/stencil-editor`).
3. **Media Context:** Медиа-файлы, аудио-записи, транскрипция, плеер (`entities/media`, `widgets/media-recording`, `widgets/media-url`).
4. **Plex Navigation Context:** Граф связей между сущностями (`widgets/plex-graph`).
5. **Workspace Context:** Оконная организация, вкладки, фокус (`entities/tab`, `app/useWindowTabs`).
6. **Settings Context:** Конфигурация приложения, темы, внешний вид (`entities/settings`, `widgets/settings`).

#### Проблемы Ubiquitous Language (Единого языка):
- В `widgets/note-editor/` файл `drawing.ts` всё еще находится в исключениях линтера `filenames.mjs` (герундий). Следует переименовать в `noteRender.ts` или `noteCanvas.ts`.
- В `widgets/plex-graph/` термины `tickets` и `standing` требуют замены на стандартные `nodeIdMap` и `focusNode`.

---

## 3. Детальный план рефакторинга крупных Composables

### Проблема:
Большие composables страдают от антипаттерна «God Composable»: принимают от 10 до 27 параметров и возвращают объекты с 15+ полями. Это усложняет тестирование, связывает независимые реактивные потоки и нарушает Single Responsibility Principle.

---

### 3.1 `src/app/useWindow.ts` (Корневой координатор окна)

#### Текущее состояние:
- Принимает глобальный контекст и возвращает объект с 15 свойствами:
  `carries, commands, doing, failure, going, held, layout, listed, log, notices, palette, places, shut, tabIcon, titled, where`.
- Смешивает управление жизненным циклом (bootstrap), горячие клавиши, палитру команд, уведомления и навигацию по вкладкам.

#### План разбиения:
Разделить `useWindow.ts` на 4 узкоспециализированных composable:
1. **`useWindowLifecycle.ts`**:
   - Отвечает строго за монтирование/размонтирование, начальный bootstrap (`starts`), слушатели изменения путей и фокуса окна.
2. **`useWindowCommandBridge.ts`**:
   - Связывает `useCommands`, вычисление `where` (CommandTarget) и палитру.
3. **`useWindowTabNavigation.ts`**:
   - Управление активными вкладками, закрытие (`shut`), переключение, отображение иконок (`tabIcon`) и заголовков (`titled`).
4. **`useWindow.ts` (Тонкий фасад)**:
   - Объединяет вышеуказанные модули без раздувания логики, возвращая сгруппированный интерфейс:
     ```ts
     return {
       tabs: { held, layout, shut, titled, tabIcon },
       commands: { palette, commands, doing, carries },
       vault: { listed, where, failure },
       notices,
       navigation: going,
     }
     ```

---

### 3.2 `src/app/useCommands.ts` (Координатор палитры команд)

#### Текущее состояние:
- Принимает **27 входных зависимостей** (`CommandsDeps`)!
- Отвечает за:
  - Реактивное состояние открытости и инпута палитры (`asked`, `palette`).
  - Поиск по заметкам (`search`).
  - Регистрацию действий для хранилищ (vaults).
  - Сценарии создания новых файлов (колод, заметок, трафаретов).
  - Хоткеи и аккорды клавиш.

#### План разбиения:
1. **`usePaletteState.ts`**:
   - Управление открытием/закрытием палитры, введенным текстом, фильтрацией и выбранным индексом.
2. **`usePaletteSearch.ts`**:
   - Поиск по заметкам, ранжирование результатов, интеграция с `searchCore`.
3. **`useVaultCommands.ts`**:
   - Сценарии открытия хранилища, создания колод/трафаретов, переименования, экспорта.
4. **`useCommands.ts`**:
   - Оркестратор палитры, агрегирующий `usePaletteState`, `usePaletteSearch` и `useVaultCommands`. Зависимости сокращаются с 27 до 3-4 сгруппированных контекстов.

---

### 3.3 `src/app/useWindowKinds.ts` (Регистратор типов вкладок)

#### Текущее состояние:
- Монолитный файл на 245 строк. Внутри одной функции создаются адаптеры всех 10 видов вкладок:
  `note`, `deck`, `stencil`, `document`, `book`, `recording`, `url`, `agent`, `files`, `plex`.
- Каждая секция настраивает слушатели событий и специфичный роутинг.

#### План разбиения:
Вынести фабрики регистраторов в отдельные модули в `src/app/kinds/`:
1. `createNoteKind.ts` — для `note`.
2. `createDeckKinds.ts` — для `deck` и `stencil`.
3. `createMediaKinds.ts` — для `recording` и `url`.
4. `createReaderKinds.ts` — для `book` и `document`.
5. `createSystemKinds.ts` — для `files`, `agent`, `plex`.
`useWindowKinds.ts` станет простым реестром, собирающим эти фабрики.

---

### 3.4 `src/widgets/note-editor/useNoteTab.ts`

#### Текущее состояние:
- Содержит логику управления редактором заметок, синхронизацию прокрутки, маркеры конфликтов, фокус и автосохранение.

#### План разбиения:
1. **`useNoteBuffer.ts`**:
   - Управление текстом, грязным состоянием (isDirty), историей правок.
2. **`useNoteAutosave.ts`**:
   - Таймеры сброса правок на диск, отслеживание конфликтов внешнего изменения (`stale`).
3. **`useNoteTab.ts`**:
   - Компактный фасад над буфером и автосохранением.

---

### 3.5 `src/widgets/file-manager/useFileTree.ts`

#### Текущее состояние:
- Совмещает в себе построение дерева файлов, драг-н-дроп, создание папок/файлов, переименование по месту и разворачивание узлов.

#### План разбиения:
1. **`useFileTreeNodes.ts`**:
   - Структура данных дерева, фильтрация, раскрытие/сворачивание папок.
2. **`useFileTreeOperations.ts`**:
   - Операции добавления, переименования и удаления файлов.
3. **`useFileTreeDragDrop.ts`**:
   - Перетаскивание элементов в дереве.
4. **`useFileTree.ts`**:
   - Композиция операций над деревом.

---

## 4. План перехода для оставшихся виджетов (FSD `components/` & `composables/`)

| Виджет | Файлы компонентов $\rightarrow$ `components/` | Файлы композблов $\rightarrow$ `composables/` |
|---|---|---|
| **`widgets/book-reader`** | `BookTab.vue`, `BookContent.vue` | `useBookTab.ts`, `useBookReader.ts` |
| **`widgets/document-viewer`** | `DocumentTab.vue` | `useDocumentTab.ts`, `useDocumentReader.ts`, `useDocumentHighlights.ts`, `useDocumentNavigation.ts`, `useDocumentViewport.ts` |
| **`widgets/file-manager`** | `FilesTab.vue`, `FileEntryRow.vue`, `NewItemPrompt.vue` | `useFilesTab.ts`, `useFileTree.ts` |
| **`widgets/note-editor`** | `NoteTab.vue` | `useNoteTab.ts` |
| **`widgets/plex-graph`** | `PlexTab.vue` | `usePlexTab.ts`, `usePlexView.ts`, `usePlexParts.ts` |
| **`widgets/preset-editor`** | `PresetTab.vue`, `preset-settings/`, `curve-slider/` | `usePresetTab.ts` |
| **`widgets/text-editor`** | `TextEditorTab.vue` | `useTextEditor.ts` |

---

## 5. Дорожная карта выполнения (Roadmap)

### Этап 1: Устранение межвиджетных зависимостей (FSD Boundary Cleanup)
1. Вынести `openNotes` и общие типы заметок из `widgets/note-editor/` в `entities/note/`.
2. Вынести пресеты расписания колод из `widgets/preset-editor/core` в `entities/deck/` или `shared/scheduling/`.
3. Убедиться, что ни один виджет не импортирует код другого виджета напрямую.

### Этап 2: Декомпозиция крупных Composables в `app/`
1. Разбить `useCommands.ts` на `usePaletteState`, `usePaletteSearch`, `useVaultCommands`.
2. Модуляризировать `useWindowKinds.ts` на специализированные фабрики.
3. Очистить `useWindow.ts` от побочной координации, сгруппировав возвращаемый интерфейс.

### Этап 3: Завершение унификации структуры срезов во всех виджетах
1. Перенести компоненты и композблы в `book-reader`, `document-viewer`, `file-manager`, `note-editor`, `plex-graph`, `preset-editor` в `components/` и `composables/`.
2. Обновить импорты во всех тестах и story-файлах.
3. Проверить отсутствие регрессий на линтерных проверках (`reach.test.mjs`, `filenames.mjs`, `composables.mjs`).

### Этап 4: Верификация и интеграция
1. Запуск полного набора unit-тестов: `npx vitest run --project unit`.
2. Запуск Storybook-тестов: `npx vitest run --project 'stories (chromium)'` и `stories (webkit)`.
3. Прогон всего набора линтеров: `node --test modules/tools/lint/*.test.mjs`.
4. Сборка всего пакета `@numen/editor`: `npm run build`.
