/* global createVideo, pickRun, createEnvironments */
/* exported writeSetting, writeDetail */
const report = window.REPORT
const scenarios = new Map(
  report.features.flatMap((feature) => feature.scenarios.map((s) => [s.id, s])),
)
const state = { search: '', query: '', status: 'all', currentId: '', environment: '' }

const SVG_NS = 'http://www.w3.org/2000/svg'
const STATUS_LABELS = { passed: 'Passed', failed: 'Failed', skipped: 'Skipped' }

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

const ICON_ATTRIBUTES = {
  viewBox: '0 0 24 24',
  fill: 'none',
  stroke: 'currentColor',
  'stroke-width': '2',
  'stroke-linecap': 'round',
  'stroke-linejoin': 'round',
}

const createSvgElement = (tag, attributes) => {
  const element = document.createElementNS(SVG_NS, tag)
  for (const [name, value] of Object.entries(attributes)) element.setAttribute(name, String(value))
  return element
}

const createIcon = (name) => {
  const svg = createSvgElement('svg', {
    ...ICON_ATTRIBUTES,
    class: `icon icon-${name}`,
    'aria-hidden': 'true',
  })
  svg.append(...window.ICONS[name].map(([tag, attributes]) => createSvgElement(tag, attributes)))
  return svg
}

const createStatusDot = (status) =>
  createElement('span', {
    class: `dot ${status}`,
    role: 'img',
    'aria-label': STATUS_LABELS[status],
  })

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

const formatRunMeta = (startedAt) => {
  const date = startedAt.toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'short' })
  return [date, report.overview.branch].filter(Boolean).join(' · ')
}

const writeStartedAt = () => {
  const startedAt = new Date(report.overview.startedAt)
  const element = document.getElementById('started-at')
  element.textContent = formatRunMeta(startedAt)
  element.title = [startedAt.toISOString(), report.overview.commit].filter(Boolean).join(' ')
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

const createLiveBadge = () => createElement('span', { class: 'live-badge' }, ['live'])

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
      createStatusDot(scenario.status),
      createElement('span', { class: 'name' }, [scenario.name]),
      ...(scenario.live ? [createLiveBadge()] : []),
      createElement('span', { class: 'time' }, [formatDuration(scenario.duration)]),
    ],
  )

const createFeatureNode = (feature, shown) => {
  const total = feature.scenarios.length
  const summary = createElement('summary', { title: feature.uri }, [
    createIcon('chevron-right'),
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

const createScenarioHeader = (scenario, environments) =>
  createElement('header', { class: 'scenario-header' }, [
    createElement('p', { class: 'crumb' }, [
      createElement('span', {}, [scenario.featureName]),
      createElement('span', { class: 'crumb-file' }, [`${scenario.featureUri}:${scenario.line}`]),
    ]),
    createElement('div', { class: 'title-row' }, [
      createElement('h2', {}, [scenario.name, ...(scenario.live ? [createLiveBadge()] : [])]),
      environments,
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

const createError = (message) =>
  createElement('div', { class: 'error' }, [
    createElement('p', { class: 'error-title' }, [createIcon('circle-alert'), 'Error']),
    createElement('pre', {}, [message]),
  ])

const createStepFailure = (step, scenario) => {
  if (step.status !== 'failed') return []
  const parts = step.errorMessage ? [createError(step.errorMessage)] : []
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

const createStep = (step, index, scenario) => {
  const isRun = step.start !== null
  return createElement(
    'li',
    {
      class: `step ${step.status}`,
      'data-index': String(index),
      title: `line ${step.line}`,
    },
    [
      createElement('span', { class: 'step-number' }, [String(index + 1)]),
      step.status === 'passed' ? createElement('span') : createStatusDot(step.status),
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
  scenario.errorMessage ? createError(scenario.errorMessage) : null

const createStepList = (scenario) =>
  createElement(
    'ol',
    { class: 'steps', 'aria-label': 'Steps' },
    scenario.steps.map((step, index) => createStep(step, index, scenario)),
  )

const writeDetail = (scenario) => {
  const run = pickRun(scenario)
  const view = { ...scenario, ...run }
  const stepList = createStepList(view)
  const video = view.video
    ? createVideo(view, stepList)
    : createElement('p', { class: 'empty' }, ['No video.'])
  document
    .getElementById('detail')
    .replaceChildren(
      ...[
        createScenarioHeader(view, createEnvironments(scenario, run)),
        video,
        createScenarioError(view),
        stepList,
      ].filter(Boolean),
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
  selectScenario(decodeURIComponent(location.hash.slice(1)) || getFirstScenarioId()),
)

document.querySelector('.search').prepend(createIcon('search'))
writeStartedAt()
writeStatusFilter()
state.currentId = decodeURIComponent(location.hash.slice(1)) || getFirstScenarioId() || ''
writeTree()
selectScenario(state.currentId)
