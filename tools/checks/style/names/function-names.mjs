// A function, method, function-valued field or callback is named with an imperative verb in the base form
// (`getTab`, `readFile`, `StatFile`) or a predicate (`isOpen`, `hasChild`, `canMove`). The first word of the
// name is checked: a participle (`Opened`), a third-person verb (`Serves`), a preposition (`After`, `At`)
// and a bare noun (`Name`, `Recipe`) say what the thing is, not what it does.

const PREDICATES = new Set(['is', 'has', 'can', 'should', 'will', 'does', 'are', 'was', 'were'])

const VERBS = new Set(`get set find list read write fetch create make build load save store open close start stop run exec execute call invoke apply update delete remove add insert append prepend push pop shift put drop clear reset init initialize new parse format render draw paint mount unmount attach detach bind unbind subscribe unsubscribe listen emit send receive publish dispatch notify watch observe track log trace warn fail assert expect verify validate check test ensure require resolve reject wait sleep delay retry cancel abort pause resume flush sync merge split join map filter reduce sort group collect gather select pick choose toggle show hide enable disable activate deactivate focus blur scroll move resize rotate scale zoom copy cut paste clone convert encode decode encrypt decrypt hash sign compress decompress serialize deserialize marshal unmarshal wrap unwrap normalize sanitize escape unescape trim pad truncate replace rename match compare diff patch import export register unregister install uninstall configure provide inject reload refresh rebuild scan walk visit traverse search query lookup locate compute calculate measure estimate derive extract expand collapse reveal capture release acquire lock unlock allow deny block grant revoke authorize authenticate login logout connect disconnect dial accept serve handle process transform translate define declare assign bump advance rewind skip take give hold keep let lose drain fill pour pull poll peek probe pong print replay restore backup recover repair fix heal unmark rank score weigh clamp round floor ceil ceiling abs redact mask quote unquote tokenize lex tokenise slice concat flatten unique dedupe distinct reverse shuffle swap exchange substitute override extend inherit implement mix compose chain curry memoize debounce throttle enqueue dequeue spawn kill terminate exit quit wake trigger fire raise throw catch panic protect secure permit forbid prohibit veto approve confirm prompt ask answer reply respond request submit navigate go back forward redirect route unlink follow unfollow relate deduce infer guess detect discover identify recognize classify categorize cluster partition bucket predict train fit embed vectorize quantize dequantize rerank generate produce yield consume discard purge prune evict expire age tick arrange organize align animate fade slide tween interpolate highlight deselect uncheck press click tap hover drag swipe pinch commit stage unstage rollback revert undo redo begin end finish complete done finalize prepare setup teardown dispose destroy cleanup clean sweep wipe reap adopt orphan borrow lend share bootstrap boot launch spin shutdown halt freeze thaw suspend lift lower grow shrink stretch squeeze flip invert negate mirror emulate simulate await ready expose nest trap intercept sift allowlist denylist whitelist blacklist reserve claim offer demand need want chmod chown mkdir rmdir touch symlink readlink glob fsync reopen refetch reindex reparse rerun rescan resync restart recompute relink zip unzip tar untar pack unpack bundle unbundle download upload sideload use contain enclose postpone elapse link place defer describe settle forget dismiss leave change step strip evaluate seek deliver try reach enter refuse transcribe minimize ignore look isolate provision edit pass silence rethrow say tell calibrate widen switch rewrite break invalidate supervise summarize retire seal admit lay remember proofread suggest announce reconcile inspect shorten jump divide relay preload compile support attempt carry review travel stand decide determine prefer prioritize populate hydrate acknowledge abandon adjust annotate assemble audit combine conceal consolidate constrain control coordinate correct debug decline decorate delegate demote deploy deprecate design display distribute duplicate eliminate enforce enrich enumerate erase establish exclude explain explore flag force free generalize guide include increment initiate instantiate integrate interpret interrupt intersect introduce iterate learn maintain manage maximize migrate modify monitor mutate negotiate omit operate optimize overflow overlap overwrite persist pin plot preserve prevent promote propagate propose prove react rebase recall recreate recycle refine reflect regenerate reinstate reorder repeat replicate represent reproduce reschedule restrict resubmit retain retrieve return reuse revalidate revise revisit rewire roll satisfy scrub segment sequence shed simplify specify stabilize stamp standardize subtract summon supply suppress synchronize tighten tolerate transfer transmit treat tune unfold unify unload unset unwind upgrade visualize vote warm withdraw withhold unlisten reconnect reclaim recheck notice swallow ping chat do print index count limit mark guard schedule plan record report sample snapshot sink fork overlay mock stub spy fake seed stat signal stream queue batch cache rate
`.split(/\s+/).filter(Boolean))

const PREPOSITIONS = new Set([
  'at', 'in', 'on', 'of', 'by', 'for', 'with', 'without', 'from', 'to', 'into', 'over', 'under', 'after', 'before',
  'until', 'since', 'per', 'via', 'as', 'within', 'between', 'across', 'around', 'through',
])

// Names the standard library or the language fixes. A Go type satisfies `fmt.Stringer`, `error`, `sort.Interface`,
// `driver.Valuer`, `driver.Stmt`, `http.ResponseWriter`, so the method has that name.
const FIXED_NAMES = new Set([
  'String', 'Error', 'Len', 'Less', 'Swap', 'Cap', 'Value', 'Err', 'Next', 'NumInput', 'Header', 'main', 'init',
  'constructor', 'render', 'setup', 'play', 'Now', 'now', 'toString', 'toJSON', 'valueOf', 'then', 'catch', 'finally', 'next', 'done',
  'Unwrap', 'Format', 'Scan', 'Printf', 'Println',
])

// Names that are still to be decided with the owner: option functions and default constructors.
const PENDING_FIRST_WORDS = new Set(['with', 'default', 'without'])

// Parameters and fields that hold a callback and are named for their role, not their action.
const CALLBACK_PARAMETER_NAMES = new Set([
  'fn', 'f', 'cb', 'callback', 'handler', 'resolve', 'reject', 'next', 'done', 'yield', 'emit', 'run', 'listener',
  'visit', 'less', 'eq', 'cmp', 'key', 'ok', 'err',
])

// A function-valued field or parameter named for its role: a handler, an observer, a listener, a callback, a loader.
const ROLE_LAST_WORDS = new Set(['handler', 'observer', 'listener', 'callback', 'loader'])

function isRoleName(name) {
  const words = getNameWords(name)
  return words.length > 0 && ROLE_LAST_WORDS.has(words[words.length - 1].toLowerCase())
}

// A Go field or parameter named `on…` that holds a callback, or a method `On…` that takes one: `onError func(error)`,
// `OnClose(fn func() error)`. `OnCard(card int) Problem` takes no callback and is checked.
const RE_GO_ON_NAME = /^[oO]n[A-Z]/
const RE_GO_FUNC_PARAMETER = /\(\s*[A-Za-z0-9_]+\s+func\(/
// A Go field or parameter holding a function with no parameters that answers one value: `plugins func() PluginCommands`.
// Like an accessor method it keeps the name of what it answers. A bool, an error and several values are not one.
const RE_GO_PROVIDER_FIELD = /^\t[A-Za-z0-9_]+\s+func\(\)\s+(?!\s|bool\b|error\b|\()[A-Za-z0-9_.*[\]]+\s*,?\s*$/
// A Vue slot is named for its place and declared as a function that returns nodes: `icon(props: { id: string }): VNode[]`.
const RE_TS_SLOT = /\)\s*:\s*VNode\[\]\s*[;,]?\s*$/
// A TypeScript member holding a function with no parameters that answers one value: `today: () => string`, `widest(): number`.
const RE_TS_PROVIDER_MEMBER = /^\s+(?:readonly\s+)?[a-z][A-Za-z0-9_$]*\??\s*(?::\s*\(\)\s*=>|\(\)\s*:)\s*(?!\s|boolean\b|void\b|never\b|Promise\b|asserts\b|[A-Za-z0-9_$.]+\s+is\b)[^;{]+;?\s*$/

const GO_TEST_PREFIX = /^(?:Test|Benchmark|Fuzz|Example)(?:[A-Z_0-9]|$)/
const CONTROL_WORDS = new Set(['if', 'for', 'while', 'switch', 'catch', 'with', 'return', 'else', 'do', 'function', 'super'])

/** The words of a camelCase or PascalCase name, in order. */
function getNameWords(name) {
  return name
    .replace(/_+/g, ' ')
    .replace(/([a-z0-9])([A-Z])/g, '$1 $2')
    .replace(/([A-Z]+)([A-Z][a-z])/g, '$1 $2')
    .split(' ')
    .filter(Boolean)
}

/**
 * The reason a function name breaks the rule, or '' when the name is fine.
 * isGo selects the language conventions: `use…` and `on…` of Vue and the DOM are fine in TypeScript only.
 */
export function getNameFault(name, isGo) {
  if (FIXED_NAMES.has(name) || CONTROL_WORDS.has(name)) return ''
  if (isGo && GO_TEST_PREFIX.test(name)) return ''
  const words = getNameWords(name)
  if (words.length === 0) return ''
  const first = words[0].toLowerCase()
  if (words[0].length === 1 && words.length > 1 && !isGo) return ''
  if (PREDICATES.has(first) || VERBS.has(first) || PENDING_FIRST_WORDS.has(first)) return ''
  if (first === 'to' || first === 'from') return ''
  if (!isGo && (first === 'use' || first === 'on' || first === 'as')) return ''
  if (name.length <= 1) return ''
  if (PREPOSITIONS.has(first)) return 'preposition'
  if (first.endsWith('ing')) return 'participle'
  if (first.endsWith('ed') || /^(?:known|shown|seen|done|gone|given|taken|kept|chosen|hidden|written|drawn|spoken|lost|left|held|built|sent)$/.test(first)) return 'participle'
  if (first.endsWith('s') && !first.endsWith('ss') && (VERBS.has(first.slice(0, -1)) || VERBS.has(first.replace(/ies$/, 'y')) || VERBS.has(first.replace(/es$/, '')))) return 'third-person'
  if (first.endsWith('s')) return 'bare-noun'
  return 'bare-noun'
}

// A Go type is a thing or a role, so its last word is a noun. A gerund (`Refreshing`) and a participle (`Listed`) name
// an action or a state. These words end in `ing` or `ed` and are nouns or adjectives. A name that starts with a command
// verb (`EnsureRegistered`, `SetTaskDone`) is a command type and may end in a participle.
const TYPE_NOUNS = new Set([
  'setting', 'heading', 'binding', 'keybinding', 'encoding', 'padding', 'warning', 'rating', 'recording', 'string',
  'thing', 'ring', 'finding', 'reading', 'reasoning', 'pending', 'spring', 'ceiling', 'king', 'wing', 'bed', 'seed', 'feed', 'speed', 'need', 'embed', 'shed', 'sled',
])
const COMMAND_VERBS = new Set(['ensure', 'set', 'mark', 'make', 'keep', 'leave'])
const IRREGULAR_PARTICIPLES = new Set(['known', 'shown', 'seen', 'done', 'gone', 'given', 'taken', 'kept', 'chosen', 'hidden', 'written', 'drawn', 'spoken', 'lost', 'left', 'held', 'built', 'sent'])

/** The reason a Go type name breaks the rule, or '' when the last word of the name is a noun. */
export function getTypeNameFault(name) {
  const words = getNameWords(name)
  if (words.length === 0) return ''
  const last = words[words.length - 1].toLowerCase()
  if (TYPE_NOUNS.has(last)) return ''
  if (last.length > 4 && last.endsWith('ing')) return 'gerund'
  const isParticiple = (last.length > 3 && last.endsWith('ed') && !last.endsWith('eed')) || IRREGULAR_PARTICIPLES.has(last)
  if (isParticiple) return words.length > 1 && COMMAND_VERBS.has(words[0].toLowerCase()) ? '' : 'participle'
  return ''
}

const RE_GO_TYPE = /^type\s+([A-Za-z0-9_]+)\b/
const RE_GO_TYPE_BLOCK_START = /^type\s*\(\s*$/
const RE_GO_TYPE_BLOCK_MEMBER = /^\t([A-Za-z0-9_]+)(?:\[[^\]]*\])?\s+\S/

/** The Go type names a line declares. state.inTypeBlock carries a `type (` block from one line to the next. */
export function findTypeNames(line, state) {
  if (RE_GO_TYPE_BLOCK_START.test(line)) {
    state.inTypeBlock = true
    return []
  }
  if (state.inTypeBlock) {
    if (/^\)/.test(line)) state.inTypeBlock = false
    const member = RE_GO_TYPE_BLOCK_MEMBER.exec(line)
    return member ? [member[1]] : []
  }
  const declared = RE_GO_TYPE.exec(line)
  return declared ? [declared[1]] : []
}

const RE_GO_FUNC = /^func\s+(?:\([^)]*\)\s*)?([A-Za-z0-9_]+)\s*[[(]/
const RE_GO_INTERFACE_START = /^type\s+[A-Za-z0-9_]+(?:\[[^\]]*\])?\s+interface\s*\{\s*$/
const RE_GO_INTERFACE_METHOD = /^\t([A-Za-z0-9_]+)\s*\(/
const RE_GO_FUNC_FIELD_LINE = /^\t([A-Za-z0-9_]+)\s+func\(/
// A Go method with no parameters that answers one value is an accessor: Go names it for what it returns, with no
// `Get` prefix. A method that answers a bool is a predicate and an error is no value, so neither is one.
const RE_GO_ACCESSOR = /^(?:func\s+\([^)]*\)\s*)?[A-Za-z0-9_]+\(\)\s+(?!bool\b|error\b|\()[A-Za-z0-9_.*[\]]+\s*(?:\{.*)?$/

const RE_TS_FUNCTION = /\bfunction\*?\s+([A-Za-z0-9_$]+)\s*[(<]/
const RE_TS_CONST_FUNCTION = /\b(?:const|let|var)\s+([A-Za-z0-9_$]+)\s*(?::[^=]+)?=\s*(?:async\s+)?(?:\([^)]*\)\s*(?::[^=]+)?=>|[A-Za-z0-9_$]+\s*=>|function\b|[A-Za-z0-9_$.]+\.bind\b)/
const RE_TS_CONST_FACTORY = /\b(?:const|let|var)\s+([A-Za-z0-9_$]+)\s*(?::[^=]+)?=\s*(?:debounce|throttle|memoize|compose|curry|once|bind)\s*(?:<[^>]*>)?\(/
const RE_TS_METHOD = /^\s+(?:(?:public|private|protected|static|async|override|readonly)\s+)*([A-Za-z0-9_$]+)\s*(?:<[^>]*>)?\([^)]*\)\s*(?::[^{=;]+)?\{\s*$/
const RE_TS_MEMBER_SIGNATURE = /^\s+(?:readonly\s+)?([a-z][A-Za-z0-9_$]*)\??\s*(?:<[^>]*>)?\([^)]*\)\s*:\s*[^;{=]+;?\s*$/
const RE_TS_MEMBER_ARROW = /^\s+(?:readonly\s+)?([a-z][A-Za-z0-9_$]*)\??\s*:\s*(?:async\s+)?(?:\([^)]*\)|[A-Za-z0-9_$]+)\s*=>\s*[A-Za-z0-9_$<>\[\]|. '"&?:]*[^,{(\s]\s*;?\s*$/

/**
 * The names a line declares that are functions: functions, methods, function-valued fields and members.
 * state.inInterface carries the Go interface block from one line to the next.
 */
/** Whether the line declares a Go accessor: a method with no parameters that answers one value. */
export function isGoAccessor(line) {
  return RE_GO_ACCESSOR.test(line.trim())
}

export function findFunctionNames(line, isGo, state) {
  const names = []
  if (isGo) {
    if (RE_GO_INTERFACE_START.test(line)) {
      state.inInterface = true
      return names
    }
    if (state.inInterface) {
      if (/^\}/.test(line)) {
        state.inInterface = false
        return names
      }
      const method = RE_GO_INTERFACE_METHOD.exec(line)
      if (method && !RE_GO_ACCESSOR.test(line.trim()) && !(RE_GO_ON_NAME.test(method[1]) && RE_GO_FUNC_PARAMETER.test(line))) {
        names.push(method[1])
      }
      return names
    }
    const declared = RE_GO_FUNC.exec(line)
    if (declared && !(declared[0].startsWith('func (') && RE_GO_ACCESSOR.test(line)) &&
      !(RE_GO_ON_NAME.test(declared[1]) && RE_GO_FUNC_PARAMETER.test(line))) {
      names.push(declared[1])
    }
    const field = RE_GO_FUNC_FIELD_LINE.exec(line)
    if (field && !RE_GO_ON_NAME.test(field[1]) && !isRoleName(field[1]) && !RE_GO_PROVIDER_FIELD.test(line)) names.push(field[1])
    return names.filter((name) => !CALLBACK_PARAMETER_NAMES.has(name))
  }
  if (RE_TS_SLOT.test(line) || RE_TS_PROVIDER_MEMBER.test(line)) return names
  for (const re of [RE_TS_FUNCTION, RE_TS_CONST_FUNCTION, RE_TS_CONST_FACTORY, RE_TS_METHOD, RE_TS_MEMBER_SIGNATURE, RE_TS_MEMBER_ARROW]) {
    const m = re.exec(line)
    if (m) {
      if (!((re === RE_TS_MEMBER_SIGNATURE || re === RE_TS_MEMBER_ARROW) && isRoleName(m[1]))) names.push(m[1])
      break
    }
  }
  return names.filter((name) => !CALLBACK_PARAMETER_NAMES.has(name) && /^[a-z_$]/.test(name))
}

const RE_TS_CONST_CREATED = /\b(?:const|let|var)\s+([a-z][A-Za-z0-9_$]*)\s*(?::[^=]+)?=\s*(?:await\s+)?create[A-Z][A-Za-z0-9_$]*\s*(?:<[^>]*>)?\(/
const RE_TS_MEMBER_NAMED_TYPE = /^\s+(?:readonly\s+)?([a-z][A-Za-z0-9_$]*)\??\s*:\s*([A-Z][A-Za-z0-9_$]*)\s*[;,]?\s*$/

/**
 * Names that may hold a function: a constant made by a `create…` call, and a member typed by a type of the same name
 * (`follows: Follows`). The caller reports one only when its fault is not a bare noun, since a value named by a noun is fine.
 */
export function findPossibleFunctionNames(line, isGo) {
  if (isGo) return []
  const created = RE_TS_CONST_CREATED.exec(line)
  if (created) return [created[1]]
  const member = RE_TS_MEMBER_NAMED_TYPE.exec(line)
  if (member && member[2].toLowerCase() === member[1].toLowerCase()) return [member[1]]
  return []
}
