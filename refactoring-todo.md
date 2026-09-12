# Numen — todo рефакторинга

Список выведен из `refactoring-remaining.md` целиком. Ветка `feat/a-book-made-for-a-screen-is-read`, состояние на коммит `bfc676bf`.

Шаг закрывается только по своему критерию. Общий прогон, обязательный после любого шага, трогающего окно, — из `modules/apps/desktop/editor`:

```
npx vitest run --project unit
npm run check --prefix modules/tools/depgraph
node --test modules/tools/lint/*.test.mjs
```

Плюс `npm run build` там, где менялись пути, и истории по одному инстансу: `vitest run --project 'stories (chromium)'`, затем `'stories (webkit)'`.

---

## 0. Решения, которые надо записать до работы

- [x] Политика для тестов и историй: **подчиняются** правилу слоёв. Исключение сделало бы дыру там, где чужое тянут чаще всего; законные случаи идут строкой в `baseline` с причиной, как уже сделано в `@numen/ui` для `Editor.stories.ts`.
- [x] Судьба стенда: вынесен из слоёв в `src/testing/`, и правила его не читают ни с одного конца.
- [x] `entities/media/kind.ts` → `entities/tab/*`: закрывается через `@x`, связь по существу.
- [x] Сегмент для DOM-адаптера настроек — `lib`, а не `api`.

---

## 1. Алиас `@/` и импорты

- [x] Завести `"paths": { "@/*": ["./src/*"] }` в `modules/apps/desktop/editor/tsconfig.json`.
- [x] Тот же алиас в `resolve.alias` у `vite.config.ts`.
- [x] Тот же алиас в `vitest.config.ts` — на верхнем уровне и в обоих проектах.
- [x] `.storybook/` отдельного алиаса не требует: `@storybook/vue3-vite` берёт `vite.config.ts`.
- [x] Переписать импорты, выходящие за свою папку, на `@/…` — 569 спецификаторов в 185 файлах, четырьмя агентами по слоям.
- [x] Убрать `'../../shared/../features/command-palette/lists'`.
- [ ] Правило в `layers.cjs`: импорт выше своей папки пишется через `@/`.

**Критерий:** `grep` находит только законные случаи — путь внутри своего среза у файла, вложенного на три уровня, и две фикстуры из `libs/protocol`, которые лежат вне `src/` и через `@/` не выражаются. `vue-tsc` чист, 2031 юнит-тест зелёный.

**Попутно вскрылось и починено:** на ветке было 25 ошибок типов, оставшихся от прошлых переносов, — `vitest` их не ловит, потому что не проверяет типы, а `npm run build` на них бы упал. Причины: реэкспорт `export type {...} from` не вносит имя в область файла; несколько модулей уехали в `entities/` и в `composables/`, а импорты остались; два разных типа `Tab` — вкладка окна и состояние открытой заметки — столкнулись в `shared/core.ts`.

---

## 2. Один `layers.cjs` с таблицей рангов

- [x] Таблица рангов: `shared` 0 без срезов, `entities` 1, `features` 2, `widgets` 3, `pages`/`screens` 4, `app`/`window` 5 без срезов.
- [x] Массив `forbidden` собирается из таблицы кодом, а не выписывается руками.
- [x] Правило «слой не тянет ранг выше своего».
- [x] Правило «срез не тянет соседний срез своего слоя».
- [x] Папка под `src/`, которой нет в таблице, считается срезом верхнего ранга — этим ловится плоский `@numen/flashcards`.
- [x] Путь `@x` не роняет прогон.
- [x] Удалить `screens.cjs`.
- [x] Убрать `boundary()` из `check.mjs`; `unscreened` переименован в `unlayered`.
- [x] `no-folder-going-round` **оставлен**, а не снят: фикстура `testdata/ring` показывает кольцо между двумя папками внутри `shared/`, которого файловое правило не видит. Перенесён в `layers.cjs`.
- [x] Тестовый стенд выведен из слоёв: `src/shared/testing/` → `src/testing/`, все правила его не читают. Он собирает целое окно, а рисуют его тесты каждого слоя.
- [x] Рёбра из `testdata/screens/` сохранены: один `layers.test.mjs` гоняет три фикстуры — `layers`, `screens`, `ring`.
- [x] Тест называет каждое ребро фикстуры и говорит, каким правилом оно отбито.
- [x] Заменить контрольный файл `@numen/editor` в `modules.mjs` на существующий.
- [x] Переписать инвариант покрытия.
- [x] Сохранить причину для `@numen/mobile`.
- [ ] Собрать `baseline` прогоном чекера — последним, когда нарушения направления закрыты.

---

## 3. Go-заставы

- [x] `go test ./container/...` в `modules/libs/core` — зелёный.
- [x] `go test ./internal/layers/...` в `modules/apps/desktop` — **был красный**: `settings_test.go` читал `editor/src/settings-tab/paths.json`, путь доФСД. Та же слепота, что у фронтового чекера. Исправлено.
- [x] Найдены и вычищены остальные ссылки на доФСД-пути в Go: комментарий в `adapter/mcp/search_test.go` называл `agent-tab/places.ts:spotOf`, которого нет.
- [x] Просмотреть семь записей `baseline` — у каждой написана причина, ни одна не мёртвая.
- [x] Следующей снимается `adapter/window/editor → container`: ведущий адаптер тянет композиционный корень, то есть окно собирает то, что само же и подаёт. Корень должен вручать ему готовое. Остальные шесть — про общий словарь, и снимаются переносом слова, а не разворотом зависимости.
- [ ] Проверить, что всё, чего приложение видеть не должно, действительно лежит под `internal/`.

**Критерий:** оба прогона зелёные, следующая снимаемая строка названа.

---

## 4. Разбор `Core` на порты

- [x] Убрать пары старое/новое имя — их оказалось двенадцать, а не одиннадцать: `opening`/`getInitialOpenPath`, `attending`/`setFocus`, `syncing`/`getSyncEnabled`, `choosesSyncing`/`setSyncEnabled`, `hanging`/`getHangingSettings`, `choosesHanging`/`setHangingSettings`, `reviewing`/`getReviewSettings`, `choosesReviewing`/`setReviewSettings`, `settings`/`getSettings`, `choosesSetting`/`updateSettings`, `settingsFile`/`getSettingsFile`.
- [x] Сделать новые имена обязательными, старые удалить вместе с вызовами — 122 места.
- [x] `writesSettingsFile` → `saveSettingsFile` везде.
- [x] Поправить заголовки тестов, называвшие старые имена.
- [ ] Разрезать `Core` на порты по доменам. Уточнение по ходу: положить их «каждый рядом со своей сущностью» нельзя в лоб — составной `Core` тогда сшивает четыре среза `entities/`, а срез не тянет соседний срез. Либо порты живут в `app/ports/`, либо каждый срез открывает свой `@x`.
- [x] Первый шаг к этому: те, кому нужен не весь `Core`, объявляют свой узкий порт сами. `widgets/note-editor/maker.ts` зовёт ровно `create` и `join` — ему и двух методов хватит.
- [x] Удалить шим `shared/note.ts` — читателей оказалось восемнадцать, а не двенадцать.
- [ ] Вынести `Core` из `shared/` и снять баррель, который реэкспортировал три модуля `entities/` целиком: из-за него 52 файла тянули словарь заметок и настроек через слой `shared`.

**Критерий:** `grep -cE "^\s+\w+\?\(" shared/core.ts` даёт ноль; интерфейса `Core` нет; `grep -r "shared/note'" src` пусто; общий прогон зелёный.

---

## 5. Межвиджетные импорты

- [x] `openNotes` и типы заметок в `entities/note/` — сделано агентом на `bfc676bf`.
- [x] Пресеты расписания в `entities/deck/presets.ts` — там же.
- [ ] Подтвердить пересчётом, что нарушений `widgets → widgets` ноль.

**Критерий:** пересчёт даёт ноль.

---

## 6. Импорты вверх по слоям

- [ ] `shared/core.ts` → `entities/settings/*`, `entities/tab/tab`: снимается переносом из шага 4.
- [ ] `shared/icons.ts` → `entities/tab/workspace`: таблица имён иконок остаётся в `shared/`, выбор иконки по вкладке переезжает в `entities/tab/`.
- [ ] `shared/note.ts` → `entities/note`: снимается удалением шима в шаге 4.
- [ ] Разрезать `entities/settings/appearance.ts` (562 строки) по четырём швам: значения домена, DOM-адаптер, представление для палитры, сценарий выбора.
- [ ] То же для `entities/settings/hanging.ts`.
- [ ] То же для `entities/settings/sync.ts`.
- [ ] Завести срез `features/settings-commands/` и перенести туда палитровую часть и сценарий.
- [ ] `entities/note/notes.quitting.test.ts` → `features/file-conflict/flushing`: решается политикой для тестов из шага 0.
- [ ] Вычеркнуть соответствующие строки из `baseline`, а не оставить их там.

**Критерий:** пересчёт даёт ноль импортов вверх по слоям; строки вычеркнуты из `baseline`.

---

## 7. Имена функций

- [ ] Написать правило рядом с `nouns.mjs`: словарь допустимых слов плюс `baseline`.
- [ ] Тест «что правило отвергает» на синтетических именах.
- [ ] Занести сегодняшний список в `baseline`.
- [ ] Переименовать глаголы 3-го лица: `carries` (5 файлов), `holds` (4), `puts` (5), `shows` (9), `stands` (3), `does`, `mends`, `begins`, `lands`, `ends`, `reaches`, `refuses`, `takes`.
- [ ] Переименовать герундии и причастия: `dressing`/`dressed`, `minting`/`minted` (должно быть `generateId`), `offered` (8 файлов), `owed`, `spined`, `shelved`, `talked`, `styling`, `sizing`, `ranging`, `keeping`, `holding`, `asking`, `answering`, `pointing`, `pressing`, `dragging`, `naming`, `making`, `closing`, `calling`.
- [ ] Поправить имена в самих линтерах: `carries` и `echoes` в `filenames.mjs`.
- [ ] Выровнять кавычки в импортах на одинарные (`entities/note/notes.ts`, `entities/deck/presets.ts` — двенадцать строк).
- [ ] Правило на кавычки в `modules/tools/lint/`.

**Критерий правила:** `node --test modules/tools/lint/*.test.mjs` зелёный. **Критерий прохода:** `baseline` пуст.

---

## 8. Публичный API срезов

- [ ] `index.ts` каждому срезу `entities/*`.
- [ ] `index.ts` каждому срезу `features/*`.
- [ ] `index.ts` каждому срезу `widgets/*`.
- [ ] Переписать импорты вглубь чужих срезов на импорты через `index.ts`.
- [ ] Включить правило «внутрь чужого среза — только через `index.ts`» в `layers.cjs`.

**Критерий:** `find src -name index.ts` даёт по одному на каждый срез; правило включено, прогон зелёный.

---

## 9. Вкладки в `pages/`

- [ ] Завести `src/pages/`.
- [ ] Перенести вкладки из `widgets/`; в `widgets/` оставить крупные блоки, которые страница собирает.
- [ ] `App.vue` собирает страницы из `pages/`.
- [ ] `app/useWindowKinds.ts` регистрирует виды из `pages/`.
- [ ] `pages` включён в таблицу рангов и правило работает.

**Критерий:** в `widgets/` не осталось ни одной вкладки; прогон зелёный. Отдельный коммит.

---

## 10. Сегменты по стандарту

- [ ] `components/` → `ui/` во всех тринадцати виджетах.
- [ ] `composables/` → `model/` во всех тринадцати.
- [ ] Адаптеры (`wire.ts`, `core.ts`) → `api/`.
- [ ] Служебный код среза → `lib/`.
- [ ] Разложить файлы, оставшиеся в корне срезов: `preset-editor` (13 файлов), `deck-editor` (8), `note-editor` (7), `file-manager` (6).
- [ ] Разложить `media-recording` — сегментов нет вовсе.
- [ ] Разложить `welcome` — то же.

**Критерий:** `components/` и `composables/` не встречаются под `src/`; в корне среза только `index.ts`, `types.ts`, `words.ts`.

---

## 11. Разбор больших срезов

- [ ] `widgets/note-editor/`: расшить пары `types.ts` / `noteTypes.ts`, `tab.ts` / `tabState.ts`.
- [ ] `features/command-palette/` (22 файла): расшить пары `list.ts` / `lists.ts`, `step.ts` / `steps.ts`.
- [ ] Разбить `useCommands.ts` на `usePaletteState`, `usePaletteSearch`, `useVaultCommands`.
- [ ] `CommandsDeps` (`deps.ts`, 248 строк) принимает сгруппированные контексты вместо двух десятков полей.
- [ ] Разбить `app/useWindowKinds.ts` (244 строки) на фабрики в `app/kinds/`.
- [ ] `entities/settings/appearance.ts` (562) — снимается шагом 6.
- [ ] `entities/media/transcript.ts` (471).
- [ ] `widgets/preset-editor/plot.ts` (394), `curve.ts` (353).
- [ ] `entities/tab/windowTabs.ts` (378), `entities/deck/presets.ts` (378).

**Критерий:** ни одного файла длиннее 250 строк в этих срезах; пары расшиты.

---

## 12. Транспорт в сборку адаптеров

- [ ] Вынести `createClient` из `entities/deck/cards.ts`.
- [ ] Из `entities/settings/theme.ts`.
- [ ] Из `entities/media/wire.ts`.
- [ ] Из двенадцати файлов `widgets/`.
- [ ] Сущности и виджеты получают порт параметром.
- [ ] Клиенты создаются только в `app/vault/clients.ts`.

**Критерий:** `grep -rl "@connectrpc\|@numen/wire" src/entities src/widgets` ничего не находит.

---

## 13. Незакрытые хвосты прошлых планов

- [ ] `widgets/text-editor/components/SettingsFileTab.vue` → `TextEditorTab.vue` (и тест рядом).
- [ ] `widgets/note-editor/drawing.ts` → `noteRender.ts`; вычеркнуть единственную строку из `baseline` в `filenames.mjs`.
- [ ] `widgets/deck-editor/drawn.ts` — того же рода.
- [ ] `widgets/preset-editor/drawn.ts` — то же.
- [ ] `features/file-conflict/flushing.ts` — то же.
- [ ] `PlexShowing` вместе с `SHOWINGS`, `ShowingDescriptor`, `showingOf`; вычеркнуть строку из `baseline` в `nouns.mjs`.

**Критерий:** оба `baseline` линтеров пусты.

---

## 14. За пределами `editor` — отдельной веткой

- [ ] `modules/libs/ui`: 27 файлов длиннее 250 строк, из них `Palette.vue` (524), `PlexNodeView.vue` (514), `Tree.vue` (511), `Face.vue` (392), `Menu.vue` (390).
- [ ] `modules/apps/desktop/flashcards`: плоский `src/` разложить по слоям, `presets.ts` (487) и `window.ts` (353) разбить, `counting.ts` и `keying.ts` переименовать.
- [ ] `flashcards` попадает под ту же таблицу рангов.

**Критерий:** в `libs/ui` нет файлов длиннее 250 строк; `flashcards` разложен по слоям и проверяется правилом.

---

## Поперёк всего, не дожидаясь очереди

### Хук

- [ ] `pre-push` в `.husky/`, гоняющий `npm run check --prefix modules/tools/depgraph` и `node --test modules/tools/lint/*.test.mjs`.

**Критерий:** попытка запушить ветку с нарушением границы останавливается локально. Проверяется тем же способом: сломать, убедиться, вернуть.

### Документация

- [ ] Раздел про слои в корневом `AGENTS.md`: две таблицы (фронт и Go), по строке на слой.
- [ ] Четыре строки правил: срез не тянет соседний срез; внутрь чужого среза только через `index.ts`; сегменты `ui`/`api`/`model`/`lib`/`config`; за пределы своей папки через `@/`.
- [ ] Строка про команды проверки.
- [ ] `CLAUDE.md` в одну строку со ссылкой на `AGENTS.md`.

**Критерий:** раздел умещается в экран; в нём нет ни одного правила, которое уже проверяет машина.

### Роли агентов

- [ ] `.claude/agents/frontend-engineer.md` — пишет TypeScript и Vue.
- [ ] `.claude/agents/go-engineer.md` — пишет Go.
- [ ] `.claude/agents/frontend-reviewer.md` — зеркало `go-reviewer`.
- [ ] `.claude/agents/architecture-reviewer.md` — только направление импортов и размещение по слоям, оба языка.
- [ ] `.claude/agents/naming-reviewer.md` — имена функций, типов, файлов.

**Критерий:** каждая роль — указатель на `AGENTS.md` плюс свои команды; ни одна не пересказывает правила.

### ADR

- [ ] Имя слоя `pages`.
- [ ] Стандартные имена сегментов.
- [ ] Разрез `Core` на порты.
- [ ] Отказ от Prettier.
- [ ] Снятие `no-folder-going-round`.

**Критерий:** по записи на каждое решение в `docs/adr`.

---

## Правила работы по списку

- Один перенос — один коммит. Механическое не смешивается со смысловым.
- Коммит держится в пределах 200–400 изменённых строк, где это возможно.
- Строка вычёркивается из `baseline` тем же коммитом, который снимает нарушение.
- Шаг без зелёного критерия не закрывается.
- Цифры в этом списке — снимок на `bfc676bf`. Перед сборкой `baseline` пересчитывать.
