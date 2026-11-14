// The page of the scenario report. data.js sets window.REPORT (the features and their scenarios)
// and window.findStepAt (timeline.ts) before this script runs.
;(() => {
  const report = window.REPORT
  const findStepAt = window.findStepAt
  const scenarios = new Map(report.features.flatMap((feature) => feature.scenarios.map((s) => [s.id, s])))
  const state = { query: '', isFailedOnly: false, currentId: '' }

  const createElement = (tag, attributes = {}, children = []) => {
    const element = document.createElement(tag)
    for (const [name, value] of Object.entries(attributes)) {
      if (value !== undefined && value !== null) element.setAttribute(name, value)
    }
    element.append(...children.filter((child) => child !== null && child !== undefined))
    return element
  }

  const formatDuration = (ms) => (ms < 1000 ? `${Math.round(ms)} ms` : `${(ms / 1000).toFixed(1)} s`)

  const formatOffset = (seconds) => {
    const minutes = Math.floor(seconds / 60)
    return `${minutes}:${(seconds - minutes * 60).toFixed(1).padStart(4, '0')}`
  }

  const createDot = (status) => createElement('span', { class: `dot ${status}`, title: status })

  const writeTotals = () => {
    const { passed, failed, skipped } = report.counts
    document.getElementById('totals').textContent =
      `${passed} passed, ${failed} failed, ${skipped} skipped · ${new Date(report.createdAt).toLocaleString()}`
  }

  const isScenarioShown = (feature, scenario) => {
    if (state.isFailedOnly && scenario.status !== 'failed') return false
    const text = `${feature.name} ${feature.uri} ${scenario.name}`.toLowerCase()
    return text.includes(state.query)
  }

  const createScenarioButton = (scenario) =>
    createElement(
      'button',
      { class: 'scenario', type: 'button', 'data-id': scenario.id, 'aria-current': String(scenario.id === state.currentId) },
      [createDot(scenario.status), createElement('span', { class: 'name' }, [scenario.name]), createElement('span', { class: 'time' }, [formatDuration(scenario.duration)])],
    )

  const createFeatureNode = (feature, shown) => {
    const { passed, failed } = feature.counts
    const summary = createElement('summary', { title: feature.uri }, [
      createDot(feature.status),
      createElement('span', { class: 'name' }, [feature.name]),
      createElement('span', { class: 'time' }, [`${passed}/${passed + failed} · ${formatDuration(feature.duration)}`]),
    ])
    const list = createElement('ul', {}, shown.map((scenario) => createElement('li', {}, [createScenarioButton(scenario)])))
    return createElement('details', { class: 'feature', open: '' }, [summary, list])
  }

  const writeTree = () => {
    const nodes = report.features.flatMap((feature) => {
      const shown = feature.scenarios.filter((scenario) => isScenarioShown(feature, scenario))
      return shown.length > 0 ? [createFeatureNode(feature, shown)] : []
    })
    const tree = document.getElementById('tree')
    tree.replaceChildren(...(nodes.length > 0 ? nodes : [createElement('p', { class: 'empty' }, ['No scenario matches.'])]))
  }

  const createDataTable = (rows) =>
    createElement('table', { class: 'data-table' }, [
      createElement('tbody', {}, rows.map((row) => createElement('tr', {}, row.map((cell) => createElement('td', {}, [cell]))))),
    ])

  const createDocString = (docString) => createElement('pre', { class: 'doc-string', 'data-media-type': docString.mediaType }, [docString.content])

  const createStepFailure = (step, scenario) => {
    if (step.status !== 'failed') return []
    const parts = step.errorMessage ? [createElement('pre', { class: 'error' }, [step.errorMessage])] : []
    if (scenario.screenshot) parts.push(createElement('img', { class: 'screenshot', src: scenario.screenshot, alt: 'The page when the step failed' }))
    return parts
  }

  const createStep = (step, index, scenario) => {
    const time = step.start === null ? 'not run' : `${formatOffset(step.start)} · ${formatDuration(step.duration)}`
    const line = createElement('div', { class: 'step-line' }, [
      createDot(step.status),
      createElement('span', { class: 'name' }, [createElement('span', { class: 'keyword' }, [step.keyword]), ` ${step.text}`]),
      createElement('span', { class: 'time' }, [time]),
    ])
    return createElement('li', { class: `step ${step.status}`, 'data-index': String(index), title: `line ${step.line}` }, [
      line,
      step.dataTable ? createDataTable(step.dataTable) : null,
      step.docString ? createDocString(step.docString) : null,
      ...createStepFailure(step, scenario),
    ])
  }

  const createFiles = (scenario) => {
    if (!scenario.trace) return createElement('p', { class: 'files' }, ['No trace.'])
    return createElement('p', { class: 'files' }, [
      createElement('a', { href: scenario.trace, download: '' }, ['trace.zip']),
      ' · open it with ',
      createElement('code', { class: 'command' }, [`npx playwright show-trace "${scenario.tracePath}"`]),
    ])
  }

  const createScenarioError = (scenario) =>
    scenario.errorMessage ? createElement('pre', { class: 'error' }, [scenario.errorMessage]) : null

  const createPlayer = (scenario) =>
    scenario.video
      ? createElement('video', { class: 'player', src: scenario.video, controls: '', preload: 'auto' })
      : createElement('p', { class: 'empty' }, ['No video.'])

  const highlightStep = (stepList, index) => {
    for (const item of stepList.children) {
      const isCurrent = Number(item.dataset.index) === index
      if (isCurrent && !item.classList.contains('current')) item.scrollIntoView({ block: 'nearest' })
      item.classList.toggle('current', isCurrent)
    }
  }

  const linkStepsToVideo = (player, stepList, scenario) => {
    const starts = scenario.steps.map((step) => step.start)
    const update = () => highlightStep(stepList, findStepAt(starts, player.currentTime))
    player.addEventListener('timeupdate', update)
    player.addEventListener('seeked', update)
    stepList.addEventListener('click', (event) => {
      const item = event.target.closest('.step')
      const start = item ? starts[Number(item.dataset.index)] : null
      if (start === null || start === undefined) return
      player.currentTime = start
      player.play().catch(() => update())
    })
  }

  const writeDetail = (scenario) => {
    const detail = document.getElementById('detail')
    const player = createPlayer(scenario)
    const stepList = createElement('ol', { class: 'steps' }, scenario.steps.map((step, index) => createStep(step, index, scenario)))
    detail.replaceChildren(
      createElement('p', { class: 'feature-name' }, [`${scenario.featureName} · ${scenario.featureUri}:${scenario.line}`]),
      createElement('h2', {}, [scenario.name, createElement('span', { class: `badge ${scenario.status}` }, [scenario.status]), createElement('span', { class: 'time' }, [formatDuration(scenario.duration)])]),
      player,
      createFiles(scenario),
      createScenarioError(scenario),
      stepList,
    )
    if (player.tagName === 'VIDEO') linkStepsToVideo(player, stepList, scenario)
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
    state.query = event.target.value.trim().toLowerCase()
    writeTree()
  })
  document.getElementById('failed-only').addEventListener('change', (event) => {
    state.isFailedOnly = event.target.checked
    writeTree()
  })
  document.getElementById('tree').addEventListener('click', (event) => {
    const button = event.target.closest('.scenario')
    if (button) location.hash = button.dataset.id
  })
  window.addEventListener('hashchange', () => selectScenario(decodeURIComponent(location.hash.slice(1))))

  writeTotals()
  state.currentId = decodeURIComponent(location.hash.slice(1)) || getFirstScenarioId() || ''
  writeTree()
  selectScenario(state.currentId)
})()
