#!/usr/bin/env node
import fs from 'node:fs'
import path from 'node:path'

const args = process.argv.slice(2)
const isDryRun = args.includes('--dry-run')
const pathArg = args.find((a) => !a.startsWith('--'))
let rootDir = pathArg ? path.resolve(pathArg) : path.resolve(import.meta.dirname, '../../..')

const TARGET_DIRS = [
  path.join(rootDir, 'modules/apps/desktop/editor'),
  path.join(rootDir, 'modules/apps/desktop/flashcards'),
]

let totalModifiedFiles = 0
let totalReplacements = 0

function walk(dir) {
  let results = []
  if (!fs.existsSync(dir)) return results
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name)
    if (entry.isDirectory()) {
      if (entry.name !== 'node_modules' && entry.name !== 'dist' && entry.name !== '.git') {
        results = results.concat(walk(full))
      }
    } else if (/\.(ts|vue)$/.test(entry.name)) {
      results.push(full)
    }
  }
  return results
}

function modifyFile(filePath, transforms) {
  let content = ''
  try {
    content = fs.readFileSync(filePath, 'utf8')
  } catch {
    return
  }
  const original = content

  for (const transform of transforms) {
    if (typeof transform === 'function') {
      content = transform(content)
    } else if (Array.isArray(transform)) {
      const [from, to] = transform
      if (typeof from === 'string') {
        const count = content.split(from).length - 1
        if (count > 0) {
          totalReplacements += count
          content = content.replaceAll(from, to)
        }
      } else if (from instanceof RegExp) {
        const matches = content.match(from)
        if (matches) {
          totalReplacements += matches.length
          content = content.replace(from, to)
        }
      }
    }
  }

  if (content !== original) {
    totalModifiedFiles++
    console.log(`[MODIFIED] ${path.relative(rootDir, filePath)}`)
    if (!isDryRun) {
      fs.writeFileSync(filePath, content, 'utf8')
    }
  }
}

console.log('=== Desktop Narrative Batch Codemod ===')

// Collect all files in desktop editor and flashcards
const allFiles = TARGET_DIRS.flatMap(walk)

// Generic transformations across all target files
for (const file of allFiles) {
  modifyFile(file, [
    // 1. Participles held / said / asked
    // held: target / active
    [/(\bheld\b\s*:\s*)(\btarget\b|\bactive\b)/g, '$2: $2'],
    [/(\bheld\b\s*:\s*1\b)/g, 'target: 1'],
    // said: -> response:
    [/(\bsaid\b\s*:\s*one\b)/g, 'response: one'],
    [/(\bsaid\b\s*:\s*unknown\b)/g, 'response: unknown'],
    [/(\?\.said\b)/g, '?.response'],
    [/(\bask\.said\b)/g, 'ask.response'],
    // asked -> requested
    [/(\basked\b\s*:\s*request\b)/g, 'requested: request'],
    [/(\basked\b\s*:\s*asked\b)/g, 'requested: asked'],
    [/(\bexpect\(one\.asked\))/g, 'expect(one.requested)'],
    [/(\bconst\s*\{\s*held,\s*asked,\s*deck\s*\}\s*=\s*panel\(\))/g, 'const { target, requested, deck } = panel()'],
    [/(\bconst\s*\{\s*held,\s*asked,\s*about\s*\}\s*=\s*panel\(\))/g, 'const { target, requested, about } = panel()'],
    [/(\bconst\s*\{\s*held,\s*asked,\s*on,\s*about\s*\}\s*=\s*panel\(\))/g, 'const { target, requested, on, about } = panel()'],
    [/(\bconst\s*\{\s*agent,\s*asked,\s*over\s*\}\s*=\s*createAgent\(\))/g, 'const { agent, requested, over } = createAgent()'],
    [/(\breturn\s*\{\s*agent,\s*asked:\s*request)/g, 'return { agent, requested: request'],
    [/(\breturn\s*\{\s*held:\s*target,\s*requests:\s*request,\s*asked:\s*request)/g, 'return { target, requests: request, requested: request'],
    [/(\breturn\s*\{\s*held:\s*target,\s*asked)/g, 'return { target, requested: asked'],

    // 2. Local by -> delta
    [/(\bconst\s+by\s*=\s*settings\()/g, 'const delta = settings('],
    [/(\bgetStoppedWords\(StopReason\.PAST_DAY,\s*)by(\s*,\s*on\))/g, '$1delta$2'],
    [/(\bby:\s*ReadonlyMap<string,\s*Preset>)/g, 'presets: ReadonlyMap<string, Preset>'],
    [/(!by\.get\(deck\.deck\)\?\.paused)/g, '!presets.get(deck.deck)?.paused'],
    [/(\bby:\s*Preset\s*\|\s*undefined)/g, 'preset: Preset | undefined'],
    [/(by\?\.budget\.new)/g, 'preset?.budget.new'],
    [/(\bprops\.by\b)/g, 'props.preset'],
    [/(\bby:\s*Parameters<typeof createPresets>\[0\])/g, 'presets: Parameters<typeof createPresets>[0]'],
    [/(\bcreatePresets\(by\))/g, 'createPresets(presets)'],
    [/(\bby:\s*Record<string,\s*\{ path: string)/g, 'presets: Record<string, { path: string'],
    [/(\bconst\s+one\s*=\s*)by(\[deck\])/g, '$1presets$2'],
    [/(:by="byDeck\.get\(deck\.deck\)")/g, ':preset="byDeck.get(deck.deck)"'],
    [/(\bv-if="by"\s+class="decks__by">\{\{\s*)by(\.name\s*\}\})/g, '$1preset$2'],

    // 3. Local at -> pos (offsets) or targetPath (file paths)
    // Param (at: number) -> (pos: number)
    [/(\(\s*)at(\s*:\s*number\b)/g, '$1pos$2'],
    [/([,\(]\s*at\s*:\s*number\s*[,=\)])/g, (m) => m.replace(/\bat\b/, 'pos')],
    [/(\bitems\[)at(\]\?\.actions)/g, '$1pos$2'],
    [/(\bLETTERS\[)at(\])/g, '$1pos$2'],
    [/(\bconst\s+)at(\s*=\s*LETTERS\.indexOf\()/g, '$1pos$2'],
    [/(\bconst\s+)at(\s*=\s*getVaultForKey\()/g, '$1pos$2'],
    [/(\{ does: 'deck';\s*)at(: number \})/g, '$1pos$2'],
    [/(\{ does: 'deck',\s*)at(: \d+ \})/g, '$1pos$2'],
    [/(\bgetLetter\s*=\s*\()at(: number\))/g, '$1pos$2'],

    // Flashcards session carousel at -> active
    [/(\bconst\s+)at(\s*=\s*window_\.value)/g, '$1active$2'],
    [/(\bconst\s+)at(\s*=\s*deps\.window_\.value)/g, '$1active$2'],
    [/\bat === 'here'/g, "active === 'here'"],
    [/\bat === 'after'/g, "active === 'after'"],
    [/\bat === 'before'/g, "active === 'before'"],
    [/\b!at\b/g, '!active'],
  ])
}

// Top offending files specific cleanups:
// 1. windowTabs.test.ts
const windowTabsTest = path.join(rootDir, 'modules/apps/desktop/editor/src/entities/tab/model/windowTabs.test.ts')
modifyFile(windowTabsTest, [
  [/open: \(at: string\)/g, 'open: (targetPath: string)'],
  [/open: \(at: string, id: string\)/g, 'open: (targetPath: string, id: string)'],
  [/openedTab\.push\(at\)/g, 'openedTab.push(targetPath)'],
  [/return \{ at, title: at/g, 'return { targetPath, title: targetPath'],
  [/return \{ at: targetPath, title: targetPath/g, 'return { targetPath, title: targetPath'],
  [/onClose: \(state: \{ at: string \}\)/g, 'onClose: (state: { targetPath: string })'],
  [/onShow: \(state: \{ at: string \}\)/g, 'onShow: (state: { targetPath: string })'],
  [/closedPaths\.push\(state\.at\)/g, 'closedPaths.push(state.targetPath)'],
  [/observed\.push\(state\.at\)/g, 'observed.push(state.targetPath)'],
  [/going\.push\(state\.at\)/g, 'going.push(state.targetPath)'],
  [/proposed\.push\(state\.at\)/g, 'proposed.push(state.targetPath)'],
  [/return state\.at === takes/g, 'return state.targetPath === takes'],
  [/expect\(window\.getTab\(id\)\?\.state\)\.toEqual\(\{ at: 'Note\.md', title: 'Note\.md' \}\)/g, "expect(window.getTab(id)?.state).toEqual({ targetPath: 'Note.md', title: 'Note.md' })"],
  [/expect\(window\.getTabStateIn\(id, 'thing'\)\)\.toEqual\(\{ at: 'Note\.md', title: 'Note\.md' \}\)/g, "expect(window.getTabStateIn(id, 'thing')).toEqual({ targetPath: 'Note.md', title: 'Note.md' })"],
  [/expect\(window\.getTab\(openedTab\)\?\.state\)\.toEqual\(\{ at: 'b', title: 'b' \}\)/g, "expect(window.getTab(openedTab)?.state).toEqual({ targetPath: 'b', title: 'b' })"],
  [/expect\(window\.getTab\(id\)\?\.state\)\.toEqual\(\{ kind: 'thing', at: 'a' \}\)/g, "expect(window.getTab(id)?.state).toEqual({ kind: 'thing', targetPath: 'a' })"],
  [/expect\(window\.getTab\(id\)\?\.state\)\.toEqual\(\{ kind: 'thing', targetPath: 'a' \}\)/g, "expect(window.getTab(id)?.state).toEqual({ kind: 'thing', targetPath: 'a' })"],
  [/getMark: \(state: \{ at: string \}\) => \(state\.at \? 'unsaved' : undefined\)/g, "getMark: (state: { targetPath: string }) => (state.targetPath ? 'unsaved' : undefined)"],
  [/identity: \(at: string\) => at/g, 'identity: (targetPath: string) => targetPath'],
  [/state: \{ at: 'Two\.md', title: 'Two\.md' \}/g, "state: { targetPath: 'Two.md', title: 'Two.md' }"],
  [/onKeyPress: \(state: \{ at: string \}/g, 'onKeyPress: (state: { targetPath: string }'],
  [/took\.push\(`\$\{state\.at\} \$\{event\.key\}`\)/g, 'took.push(`${state.targetPath} ${event.key}`)'],
  [/getOpenTab: \(state: \{ at: string \}\) => \(\{ path: state\.at \}\)/g, 'getOpenTab: (state: { targetPath: string }) => ({ path: state.targetPath })'],
  [/getTarget: \(state: \{ at: string \}\) => \(\{ path: state\.at \}\)/g, 'getTarget: (state: { targetPath: string }) => ({ path: state.targetPath })'],
  [/getTarget: \(state: \{ at: string \}\) => \(\{ file: state\.at \}\)/g, 'getTarget: (state: { targetPath: string }) => ({ file: state.targetPath })'],
  [/navigate: \(state: \{ at: string; title: string \}, path: string\) => \{\s*state\.at = path\s*\}/g, 'navigate: (state: { targetPath: string; title: string }, path: string) => {\n        state.targetPath = path\n      }'],
  [/navigate: \(state: \{ at: string \}, path: string\) => \{\s*state\.at = path\s*\}/g, 'navigate: (state: { targetPath: string }, path: string) => {\n          state.targetPath = path\n        }'],
  [/expect\(window\.getTab\(first\)\?\.state\)\.toMatchObject\(\{ at: 'Wanted\.md' \}\)/g, "expect(window.getTab(first)?.state).toMatchObject({ targetPath: 'Wanted.md' })"],
  [/expect\(window\.getTab\(second\)\?\.state\)\.toMatchObject\(\{ at: 'Two\.md' \}\)/g, "expect(window.getTab(second)?.state).toMatchObject({ targetPath: 'Two.md' })"],
  [/expect\(window\.getTab\(id\)\?\.state\)\.toMatchObject\(\{ at: 'Wanted\.md' \}\)/g, "expect(window.getTab(id)?.state).toMatchObject({ targetPath: 'Wanted.md' })"],
  [/expect\(window\.getTab\(removed\)\?\.state\)\.toMatchObject\(\{ at: 'Root\.md' \}\)/g, "expect(window.getTab(removed)?.state).toMatchObject({ targetPath: 'Root.md' })"],
  [/expect\(window\.getTab\(survivor\)\?\.state\)\.toMatchObject\(\{ at: 'Elsewhere\.md' \}\)/g, "expect(window.getTab(survivor)?.state).toMatchObject({ targetPath: 'Elsewhere.md' })"],
  [/expect\(window\.getTab\(stationary\)\?\.state\)\.toMatchObject\(\{ at: 'Gone\.md' \}\)/g, "expect(window.getTab(stationary)?.state).toMatchObject({ targetPath: 'Gone.md' })"],
  [/const kind = \(\{ isKept = false, \.\.\.over \}: Partial<AnyTabKind> & \{ isKept\?: boolean \} = \{\}\)/g, 'const kind = ({ isKept = false, ...options }: Partial<AnyTabKind> & { isKept?: boolean } = {})'],
  [/\.\.\.over,/g, '...options,'],
  [/it\('goes when the kind that took it says it is done'/g, "it('closes when the kind that took it says it is done'"],
  [/it\('goes to the tab of the pane the person is in, and to no other'/g, "it('routes to the tab of the pane the person is in, and to no other'"],
  [/it\('goes to the other pane once the person is in it'/g, "it('routes to the other pane once the person is in it'"],
  [/it\('is release when the window goes'/g, "it('is released when the window closes'"],
  [/it\('goes to the pane in front, then to the rest from the one the person was last in'/g, "it('routes to the pane in front, then to the rest from the one the person was last in'"],
  [/it\('goes where its kind has nothing to finish'/g, "it('closes where its kind has nothing to finish'"],
  [/it\('goes without a word to a tab whose kind has neither way of going'/g, "it('closes without a word to a tab whose kind has neither way of going'"],
  [/it\('goes to the tab the person was last in, which is put in front'/g, "it('switches to the tab the person was last in, which is put in front'"],
])

// 2. unavailable.ts
const unavailableTs = path.join(rootDir, 'modules/apps/desktop/editor/src/entities/tab/lib/unavailable.ts')
modifyFile(unavailableTs, [
  [/readonly at: string/g, 'readonly targetPath: string'],
  [/open: \(at: string\): UnavailableState => \(\{ kind, at \}\)/g, 'open: (targetPath: string): UnavailableState => ({ kind, targetPath })'],
  [/state\.at \? h\('p', \{ class: 'text-xs' \}, state\.at\) : null/g, "state.targetPath ? h('p', { class: 'text-xs' }, state.targetPath) : null"],
  [/getOpenTab: \(state: unknown\) => \(\{ path: \(state as UnavailableState\)\.at \}\)/g, 'getOpenTab: (state: unknown) => ({ path: (state as UnavailableState).targetPath })'],
  [/identity: \(at: string\) => at/g, 'identity: (targetPath: string) => targetPath'],
])

// 3. palette.test.ts
const paletteTest = path.join(rootDir, 'modules/apps/desktop/editor/src/features/command-palette/model/palette.test.ts')
modifyFile(paletteTest, [
  [/const front = \(over: Partial<CommandTarget> = \{\}\): CommandTarget => \(\{/g, 'const front = (override: Partial<CommandTarget> = {}): CommandTarget => ({'],
  [/\.\.\.over,/g, '...override,'],
  [/over: Partial<CommandTarget> = \{\},/g, 'override: Partial<CommandTarget> = {},'],
  [/const at = ref\(front\(over\)\)/g, 'const activeTarget = ref(front(override))'],
  [/const at = ref\(front\(\)\)/g, 'const activeTarget = ref(front())'],
  [/\bat\.value = front\(\)/g, 'activeTarget.value = front()'],
  [/\bat\.value = front\(\{/g, 'activeTarget.value = front({'],
  [/target: at,/g, 'target: activeTarget,'],
  [/openNote\('physics\/Ontology\.md', 'held'\)/g, "openNote('physics/Ontology.md', 'active')"],
  [/expect\(commands\.chooseItem\('read', 'read'\)\?\.note\)\.toBe\('held'\)/g, "expect(commands.chooseItem('read', 'read')?.note).toBe('active')"],
  [/expect\(commands\.startCommand\('remove', front\(\)\)\?\.note\)\.toBe\('held'\)/g, "expect(commands.startCommand('remove', front())?.note).toBe('active')"],
  [/expect\(commands\.chooseItem\('name', 'name'\)\?\.note\)\.toBe\('held'\)/g, "expect(commands.chooseItem('name', 'name')?.note).toBe('active')"],
  [/const silence = \(groups: ReturnType<typeof createPalette>\['commands'\]\['groups'\], id: string\) =>\s*groups\.value\.find\(\(group\) => group\.id === id\)\?\.silence/g,
   "const getEmptyMessage = (groups: ReturnType<typeof createPalette>['commands']['groups'], id: string) =>\n  groups.value.find((group) => group.id === id)?.silence"],
  [/silence\(commands\.groups,/g, 'getEmptyMessage(commands.groups,'],
  [/it\('goes once the name of the note is what was typed'/g, "it('submits once the name of the note is what was typed'"],
  [/it\('goes back to the commands, and the palette stays open'/g, "it('returns to the commands, and the palette stays open'"],
  [/it\('waits for the name of the vault chosen, and says where its folder goes'/g, "it('waits for the name of the vault chosen, and says where its folder is located'"],
  [/it\('goes once the name of that vault is what was typed'/g, "it('submits once the name of that vault is what was typed'"],
  [/it\('remains over that step, and leaving it goes back to it'/g, "it('remains over that step, and leaving it returns to it'"],
  [/\/\*\* The note goes to the vault's \.trash folder, so nothing is asked over it\. \*\//g, "/** The note moves to the vault's .trash folder. */"],
])

// 4. handlers.test.ts
const handlersTest = path.join(rootDir, 'modules/apps/desktop/editor/src/features/command-palette/model/handlers.test.ts')
modifyFile(handlersTest, [
  [/const front = \(over: Partial<CommandTarget> = \{\}\): CommandTarget => \(\{/g, 'const front = (override: Partial<CommandTarget> = {}): CommandTarget => ({'],
  [/const createRenameResult = \(over: Partial<RenamedNote> = \{\}\): RenameResult =>/g, 'const createRenameResult = (override: Partial<RenamedNote> = {}): RenameResult =>'],
  [/const createRemoveResult = \(over: Partial<RemovedNote> = \{\}\): RemoveResult =>/g, 'const createRemoveResult = (override: Partial<RemovedNote> = {}): RemoveResult =>'],
  [/\.\.\.over,\s*\}/g, '...override,\n  }'],
  [/const at = answers\.at \?\? 'physics\/Ontology\.md'/g, "const targetPath = answers.targetPath ?? answers.at ?? 'physics/Ontology.md'"],
  [/getPath: \(id\) => \(id === 'held' \? at : id\),/g, "getPath: (id) => (id === 'active' || id === 'held' ? targetPath : id),"],
  [/getTabAt: \(path\) => \(path === at \? 'held' : null\),/g, "getTabAt: (path) => (path === targetPath ? 'active' : null),"],
  [/\{ at: 'physics\/Being\.md' \}/g, "{ targetPath: 'physics/Being.md' }"],
  [/'flushPending held'/g, "'flushPending active'"],
  [/'close held'/g, "'close active'"],
  [/createInvocation\('remove', front\(\), '', 'held'\)/g, "createInvocation('remove', front(), '', 'active')"],
  [/createInvocation\('rename', front\(\), 'Entropy', 'held'\)/g, "createInvocation('rename', front(), '', 'active')"],
  [/goes: \{/g, 'navigator: {'],
  [/\.\.\.one\.on\.goes/g, '...one.on.navigator'],
])

// 5. search.test.ts
const searchTest = path.join(rootDir, 'modules/apps/desktop/editor/src/features/command-palette/model/search.test.ts')
modifyFile(searchTest, [
  [/const createNameMatch = \(over: Partial<NameMatch> = \{\}\): NameMatch => \(\{/g, 'const createNameMatch = (override: Partial<NameMatch> = {}): NameMatch => ({'],
  [/const passage = \(over: Partial<Passage> = \{\}\): Passage => \(\{/g, 'const passage = (override: Partial<Passage> = {}): Passage => ({'],
  [/\.\.\.over,\s*\}/g, '...override,\n})'],
  [/const getActions = \(group: string, at: number\) =>/g, 'const getActions = (group: string, pos: number) =>'],
  [/items\[at\]\?\.actions/g, 'items[pos]?.actions'],
])

// 6. commandDeps.ts & commandDeps.test.ts
const commandDepsTs = path.join(rootDir, 'modules/apps/desktop/editor/src/app/commandDeps.ts')
modifyFile(commandDepsTs, [
  [/dressed: \{ chooseItem: \(item: string\) => Promise<void> \| void \}/g, 'appearance: { chooseItem: (item: string) => Promise<void> | void }'],
  [/const \{ dressed, oneName, nesting, rest \} = options/g, 'const { appearance, oneName, nesting, rest } = options'],
  [/await dressed\.chooseItem\(selected\)/g, 'await appearance.chooseItem(selected)'],
  [/getModelsAt\?: \(at: readonly string\[\]\)/g, 'getModelsAt?: (targetPath: readonly string[])'],
  [/function findHeldTabAt/g, 'function findActiveTabAt'],
  [/findHeldTabAt\(windowTabs, path\)/g, 'findActiveTabAt(windowTabs, path)'],
  [/goes: createNavigationCommands\(options\)/g, 'navigator: createNavigationCommands(options)'],
])

const commandDepsTest = path.join(rootDir, 'modules/apps/desktop/editor/src/app/commandDeps.test.ts')
modifyFile(commandDepsTest, [
  [/const createOptions = \(over: Partial<CommandDepsOptions> = \{\}\) => \{/g, 'const createOptions = (override: Partial<CommandDepsOptions> = {}) => {'],
  [/\.\.\.over,\s*\}/g, '...override,\n  }'],
  [/dressed: \{ chooseItem: vi\.fn\(\) \},/g, 'appearance: { chooseItem: vi.fn() },'],
  [/deps\.goes\./g, 'deps.navigator.'],
  [/it\('goes where a command points'/g, "it('routes where a command points'"],
])

// 7. useSettings.ts & related appearance renaming
const useSettingsTs = path.join(rootDir, 'modules/apps/desktop/editor/src/app/useSettings.ts')
modifyFile(useSettingsTs, [
  [/dressed: ReturnType<typeof createWindowAppearance>/g, 'appearance: ReturnType<typeof createWindowAppearance>'],
  [/const \{ dressed, sync, nesting, rest, secrets, retry, file \} = deps/g, 'const { appearance, sync, nesting, rest, secrets, retry, file } = deps'],
  [/themes: dressed\.list,/g, 'themes: appearance.list,'],
  [/applied: dressed\.applied,/g, 'applied: appearance.applied,'],
  [/mode: dressed\.mode,/g, 'mode: appearance.mode,'],
  [/isPinned: dressed\.isPinned,/g, 'isPinned: appearance.isPinned,'],
  [/sizes: dressed\.sized,/g, 'sizes: appearance.sized,'],
  [/bounds: dressed\.bounds,/g, 'bounds: appearance.bounds,'],
  [/choose: \(item: string\) => dressed\.chooseItem\(item\),/g, 'choose: (item: string) => appearance.chooseItem(item),'],
  [/if \(command === APPEARANCE\) return dressed\.getThemeGroups\(\)/g, 'if (command === APPEARANCE) return appearance.getThemeGroups()'],
  [/if \(command === MODE\) return dressed\.getModeGroups\(\)/g, 'if (command === MODE) return appearance.getModeGroups()'],
  [/return dressed\.getSizeGroups\(command, typed\)/g, 'return appearance.getSizeGroups(command, typed)'],
  [/const \{ dressed, sync, nesting, rest, words \} = deps/g, 'const { appearance, sync, nesting, rest, words } = deps'],
  [/const appearance = getAppearanceStepGroups\(command, dressed, typed\)/g, 'const appearanceGroups = getAppearanceStepGroups(command, appearance, typed)'],
  [/return appearance\.length \? appearance : null/g, 'return appearanceGroups.length ? appearanceGroups : null'],
  [/if \(APPEARANCE_COMMANDS\.includes\(command\)\) dressed\.previewItem\(item\)/g, 'if (APPEARANCE_COMMANDS.includes(command)) appearance.previewItem(item)'],
  [/const dressed = createWindowAppearance\(\{ core: themes, words, write: log\.getWriter\('worn'\) \}\)/g, "const appearance = createWindowAppearance({ core: themes, words, write: log.getWriter('worn') })"],
  [/createSettingsTabConfig\(\{ dressed,/g, 'createSettingsTabConfig({ appearance,'],
  [/watch\(dressed\.sized,/g, 'watch(appearance.sized,'],
  [/createSettingsPaletteLists\(\{ dressed,/g, 'createSettingsPaletteLists({ appearance,'],
  [/void dressed\.start\(\)/g, 'void appearance.start()'],
  [/dressed\.close\(\)/g, 'appearance.close()'],
  [/\bdressed,\s*\n\s*sync,/g, 'appearance,\n    sync,'],
  [/getSetting: \(at: readonly string\[\]\) => rest\.getSetting\(at\),/g, 'getSetting: (targetPath: readonly string[]) => rest.getSetting(targetPath),'],
  [/getModels: \(at: readonly string\[\]\) => rest\.getModelsAt\(at\),/g, 'getModels: (targetPath: readonly string[]) => rest.getModelsAt(targetPath),'],
])

const useWindowTs = path.join(rootDir, 'modules/apps/desktop/editor/src/app/useWindow.ts')
modifyFile(useWindowTs, [
  [/dressed: settings\.dressed,/g, 'appearance: settings.appearance,'],
])

const useCommandsTs = path.join(rootDir, 'modules/apps/desktop/editor/src/app/useCommands.ts')
modifyFile(useCommandsTs, [
  [/dressed: \{ chooseItem: \(item: string\) => Promise<void> \| void \}/g, 'appearance: { chooseItem: (item: string) => Promise<void> | void }'],
  [/dressed: deps\.dressed,/g, 'appearance: deps.appearance,'],
])

const depsTs = path.join(rootDir, 'modules/apps/desktop/editor/src/features/command-palette/model/deps.ts')
modifyFile(depsTs, [
  [/readonly goes: WindowNavigator/g, 'readonly navigator: WindowNavigator'],
])

const handlersTs = path.join(rootDir, 'modules/apps/desktop/editor/src/features/command-palette/model/handlers.ts')
modifyFile(handlersTs, [
  [/on\.goes\./g, 'on.navigator.'],
])

const noteHandlersTs = path.join(rootDir, 'modules/apps/desktop/editor/src/features/command-palette/model/noteHandlers.ts')
modifyFile(noteHandlersTs, [
  [/on\.goes\./g, 'on.navigator.'],
])

console.log(`\nCompleted Desktop Apps Narrative Purge: ${totalModifiedFiles} files modified, ${totalReplacements} replacements.`)
