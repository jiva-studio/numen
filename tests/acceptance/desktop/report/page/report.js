/* global createVideo, playback */
/* exported writeSetting */
const report = window.REPORT
const scenarios = new Map(
  report.features.flatMap((feature) => feature.scenarios.map((s) => [s.id, s])),
)
const state = { search: '', query: '', status: 'all', currentId: '' }

const SVG_NS = 'http://www.w3.org/2000/svg'
const ICONS = {
  check: 'M3.5 8.5l3 3 6-7',
  cross: 'M4.5 4.5l7 7M11.5 4.5l-7 7',
  dash: 'M4.5 8h7',
  link: 'M6.8 9.2l2.4-2.4M7.3 4.6l1-1a2.6 2.6 0 013.7 3.7l-1 1M8.7 11.4l-1 1a2.6 2.6 0 01-3.7-3.7l1-1',
  play: 'M5 3.2v9.6L12.6 8z',
  pause: 'M5.5 3.5v9M10.5 3.5v9',
  expand: 'M2.5 6V2.5H6M10 2.5h3.5V6M13.5 10v3.5H10M6 13.5H2.5V10',
  shrink: 'M6 2.5V6H2.5M13.5 6H10V2.5M10 13.5V10h3.5M2.5 10H6v3.5',
  chevron: 'M6 4l4 4-4 4',
}
const STATUS_ICONS = { passed: 'check', failed: 'cross', skipped: 'dash' }

const writeSetting = (key, value) => {
  try {
    localStorage.setItem(key, value)
    return true
  } catch {
    return false
  }
}

const createElement = (tag, attributes = {}, children = []) => {
  const element = document.createElement(tag)
  for (const [name, value] of Object.entries(attributes)) {
    if (value !== undefined && value !== null) element.setAttribute(name, value)
  }
  element.append(...children.filter((child) => child !== null && child !== undefined))
  return element
}

const createIcon = (name) => {
  const svg = document.createElementNS(SVG_NS, 'svg')
  svg.setAttribute('class', 'icon')
  svg.setAttribute('viewBox', '0 0 16 16')
  svg.setAttribute('aria-hidden', 'true')
  const path = document.createElementNS(SVG_NS, 'path')
  path.setAttribute('d', ICONS[name])
  svg.append(path)
  return svg
}

const createStatusIcon = (status) =>
  createElement('span', { class: `status-icon ${status}`, role: 'img', 'aria-label': status }, [
    createIcon(STATUS_ICONS[status]),
  ])

const formatDuration = (ms) => (ms < 1000 ? `${Math.round(ms)} ms` : `${(ms / 1000).toFixed(1)} s`)

const formatOffset = (seconds) => {
  const minutes = Math.floor(seconds / 60)
  return `${minutes}:${(seconds - minutes * 60).toFixed(1).padStart(4, '0')}`
}

const formatCount = (count, noun) => `${count} ${noun}${count === 1 ? '' : 's'}`

const STATUS_FILTERS = [
  { status: 'all', label: 'All' },
  { status: 'passed', label: 'Passed' },
  { status: 'failed', label: 'Failed' },
  { status: 'skipped', label: 'Skipped' },
]

const writeStartedAt = () => {
  const startedAt = new Date(report.overview.startedAt)
  const element = document.getElementById('started-at')
  element.textContent = startedAt.toLocaleString(undefined, {
    dateStyle: 'medium',
    timeStyle: 'short',
  })
  element.title = startedAt.toISOString()
}

const getFilterCount = (status) =>
  status === 'all' ? report.overview.scenarios : report.overview.counts[status]

const createFilterButton = ({ status, label }) => {
  const count = getFilterCount(status)
  return createElement(
    'button',
    {
      type: 'button',
      class: count === 0 ? 'is-zero' : null,
      'data-status': status,
      'aria-pressed': String(status === state.status),
    },
    [label, createElement('span', { class: 'filter-count' }, [String(count)])],
  )
}

const writeStatusFilter = () => {
  const filters = STATUS_FILTERS.filter(
    ({ status }) => status !== 'skipped' || getFilterCount(status) > 0,
  )
  document.getElementById('status-filter').replaceChildren(...filters.map(createFilterButton))
}

const selectStatus = (status) => {
  state.status = status
  for (const button of document.querySelectorAll('.status-filter button')) {
    button.setAttribute('aria-pressed', String(button.dataset.status === status))
  }
  writeTree()
}

const isScenarioShown = (feature, scenario) => {
  if (state.status !== 'all' && scenario.status !== state.status) return false
  const text = `${feature.name} ${feature.uri} ${scenario.name}`.toLowerCase()
  return text.includes(state.query)
}

const createScenarioButton = (scenario) =>
  createElement(
    'button',
    {
      class: 'scenario',
      type: 'button',
      'data-id': scenario.id,
      'aria-current': String(scenario.id === state.currentId),
    },
    [
      createStatusIcon(scenario.status),
      createElement('span', { class: 'name' }, [scenario.name]),
      createElement('span', { class: 'time' }, [formatDuration(scenario.duration)]),
    ],
  )

const createFeatureNode = (feature, shown) => {
  const total = feature.scenarios.length
  const summary = createElement('summary', { title: feature.uri }, [
    createStatusIcon(feature.status),
    createElement('span', { class: 'name' }, [feature.name]),
    createElement('span', { class: 'count', title: formatCount(total, 'scenario') }, [
      String(total),
    ]),
  ])
  const list = createElement(
    'ul',
    {},
    shown.map((scenario) => createElement('li', {}, [createScenarioButton(scenario)])),
  )
  return createElement('details', { class: 'feature', open: '' }, [summary, list])
}

const getEmptyTreeText = () => {
  if (state.search) return `Nothing matches “${state.search}”`
  return state.status === 'all' ? 'No scenarios' : `No ${state.status} scenarios`
}

const createEmptyTree = () =>
  createElement('p', { class: 'tree-empty', role: 'status' }, [getEmptyTreeText()])

const writeTree = () => {
  const nodes = report.features.flatMap((feature) => {
    const shown = feature.scenarios.filter((scenario) => isScenarioShown(feature, scenario))
    return shown.length > 0 ? [createFeatureNode(feature, shown)] : []
  })
  const tree = document.getElementById('tree')
  tree.replaceChildren(...(nodes.length > 0 ? nodes : [createEmptyTree()]))
}

const getScenarioLink = (id) => `${location.href.split('#')[0]}#${encodeURIComponent(id)}`

const showCopied = (note, text) => {
  note.textContent = text
  note.classList.add('is-shown')
  clearTimeout(note.timer)
  note.timer = setTimeout(() => note.classList.remove('is-shown'), 1400)
}

const copyLink = async (id, note) => {
  try {
    await navigator.clipboard.writeText(getScenarioLink(id))
    showCopied(note, 'Copied')
  } catch {
    showCopied(note, 'Could not copy')
  }
}

const createActions = (scenario) => {
  const note = createElement('span', { class: 'copied', role: 'status' })
  const copy = createElement(
    'button',
    { class: 'icon-button', type: 'button', title: 'Copy link', 'aria-label': 'Copy link' },
    [createIcon('link')],
  )
  copy.addEventListener('click', () => copyLink(scenario.id, note))
  return createElement('div', { class: 'actions' }, [copy, note])
}

const createScenarioHeader = (scenario) =>
  createElement('header', { class: 'scenario-header' }, [
    createElement(
      'p',
      { class: 'feature-name', title: `${scenario.featureUri}:${scenario.line}` },
      [scenario.featureName],
    ),
    createElement('div', { class: 'title-row' }, [
      createElement('h2', {}, [
        createStatusIcon(scenario.status),
        createElement('span', {}, [
          scenario.name,
          ' ',
          createElement('span', { class: 'time' }, [formatDuration(scenario.duration)]),
        ]),
      ]),
      createActions(scenario),
    ]),
  ])

const createDataTable = (rows) =>
  createElement('div', { class: 'table-scroll' }, [
    createElement('table', { class: 'data-table' }, [
      createElement(
        'tbody',
        {},
        rows.map((row) =>
          createElement(
            'tr',
            {},
            row.map((cell) => createElement('td', {}, [cell])),
          ),
        ),
      ),
    ]),
  ])

const createDocString = (docString) =>
  createElement('pre', { class: 'doc-string', 'data-media-type': docString.mediaType }, [
    docString.content,
  ])

const createStepFailure = (step, scenario) => {
  if (step.status !== 'failed') return []
  const parts = step.errorMessage
    ? [createElement('pre', { class: 'error' }, [step.errorMessage])]
    : []
  if (scenario.screenshot)
    parts.push(
      createElement('img', {
        class: 'screenshot',
        src: scenario.screenshot,
        alt: 'The page when the step failed',
      }),
    )
  return parts
}

const createStep = (step, index, scenario, setupLength) => {
  const isRun = step.start !== null
  return createElement(
    'li',
    {
      class: `step ${step.status}${index < setupLength ? ' is-setup' : ''}`,
      'data-index': String(index),
      title: `line ${step.line}`,
    },
    [
      createStatusIcon(step.status),
      createElement('span', { class: 'keyword' }, [step.keyword]),
      createElement('span', { class: 'step-text' }, [step.text]),
      createElement(
        'span',
        { class: 'step-time', title: isRun ? `at ${formatOffset(step.start)}` : null },
        [isRun ? formatDuration(step.duration) : 'not run'],
      ),
      step.dataTable ? createDataTable(step.dataTable) : null,
      step.docString ? createDocString(step.docString) : null,
      ...createStepFailure(step, scenario),
    ],
  )
}

const createScenarioError = (scenario) =>
  scenario.errorMessage ? createElement('pre', { class: 'error' }, [scenario.errorMessage]) : null

const setSetupOpen = (stepList, isOpen) => {
  stepList.classList.toggle('is-setup-open', isOpen)
  const toggle = stepList.querySelector('.setup-toggle')
  toggle.setAttribute('aria-expanded', String(isOpen))
  toggle.setAttribute('aria-label', isOpen ? 'Hide the setup steps' : 'Show the setup steps')
}

const createSetupRow = (setup) => {
  const toggle = createElement('button', { class: 'setup-toggle', type: 'button' }, [
    createIcon('chevron'),
  ])
  toggle.addEventListener('click', (event) => {
    const stepList = event.currentTarget.closest('.steps')
    setSetupOpen(stepList, !stepList.classList.contains('is-setup-open'))
  })
  const status = playback.getWorstStatus(setup.map((step) => step.status))
  const ran = setup.filter((step) => step.start !== null)
  const duration = ran.reduce((total, step) => total + step.duration, 0)
  return createElement('li', { class: `setup-row ${status}` }, [
    createStatusIcon(status),
    createElement('span', { class: 'keyword' }, ['Setup']),
    createElement('span', { class: 'step-text' }, [formatCount(setup.length, 'step'), toggle]),
    createElement('span', { class: 'step-time' }, [
      ran.length > 0 ? formatDuration(duration) : 'not run',
    ]),
  ])
}

const createStepList = (scenario) => {
  const setupLength = playback.getSetupLength(scenario.steps)
  const items = scenario.steps.map((step, index) => createStep(step, index, scenario, setupLength))
  if (setupLength === 0) return createElement('ol', { class: 'steps' }, items)
  const setup = scenario.steps.slice(0, setupLength)
  const stepList = createElement('ol', { class: 'steps' }, [createSetupRow(setup), ...items])
  setSetupOpen(
    stepList,
    setup.some((step) => step.status === 'failed'),
  )
  return stepList
}

const writeDetail = (scenario) => {
  const stepList = createStepList(scenario)
  const video = scenario.video
    ? createVideo(scenario, stepList)
    : createElement('p', { class: 'empty' }, ['No video.'])
  document
    .getElementById('detail')
    .replaceChildren(
      ...[createScenarioHeader(scenario), video, createScenarioError(scenario), stepList].filter(
        Boolean,
      ),
    )
}

const selectScenario = (id) => {
  const scenario = scenarios.get(id)
  if (!scenario) return
  state.currentId = id
  for (const button of document.querySelectorAll('.scenario')) {
    button.setAttribute('aria-current', String(button.dataset.id === id))
  }
  writeDetail(scenario)
}

const getFirstScenarioId = () => {
  const all = [...scenarios.values()]
  return (all.find((scenario) => scenario.status === 'failed') ?? all[0])?.id
}

document.getElementById('search').addEventListener('input', (event) => {
  state.search = event.target.value.trim()
  state.query = state.search.toLowerCase()
  writeTree()
})
document.getElementById('status-filter').addEventListener('click', (event) => {
  const button = event.target.closest('button')
  if (button) selectStatus(button.dataset.status)
})
document.getElementById('tree').addEventListener('click', (event) => {
  const button = event.target.closest('.scenario')
  if (button) location.hash = button.dataset.id
})
window.addEventListener('hashchange', () =>
  selectScenario(decodeURIComponent(location.hash.slice(1))),
)

writeStartedAt()
writeStatusFilter()
state.currentId = decodeURIComponent(location.hash.slice(1)) || getFirstScenarioId() || ''
writeTree()
selectScenario(state.currentId)
