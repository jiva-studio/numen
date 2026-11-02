// Answers the calls that read the markup of a chapter after window.BACKEND_DELAY_MS, as a large book
// and a busy machine do, so a chapter is drawn between two presses of a rapid run.
{
  const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms))
  const fetchNow = window.fetch.bind(window)
  window.fetch = async (...args) => {
    const first = args[0]
    const address = typeof first === 'string' ? first : (first?.url ?? String(first))
    if (address.includes('ReadBookMarkup')) await sleep(window.BACKEND_DELAY_MS)
    return fetchNow(...args)
  }
}
