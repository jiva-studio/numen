import { execFileSync } from 'node:child_process'

export const DEFAULT_GIST_ID = 'd232b23b1b45234f6fd6570b93f2e63a'

/**
 * Updates a badge JSON file in the public GitHub Gist.
 *
 * @param {string} filename - e.g. 'coverage.json' or 'mutation.json'
 * @param {object} payload - Shields.io endpoint object: { schemaVersion: 1, label, message, color }
 */
export const updateBadgeGist = async (filename, payload) => {
  const contentStr = JSON.stringify(payload, null, 2) + '\n'
  const token = process.env.GIST_TOKEN || process.env.GH_TOKEN || process.env.GITHUB_TOKEN
  const gistId = process.env.GIST_ID || DEFAULT_GIST_ID

  if (!token) {
    console.log(`[badge] No token provided; skipping remote update for ${filename}.`)
    return
  }

  try {
    const input = JSON.stringify({
      files: {
        [filename]: {
          content: contentStr,
        },
      },
    })

    execFileSync('gh', ['api', '--method', 'PATCH', `/gists/${gistId}`, '--input', '-'], {
      input,
      encoding: 'utf8',
      env: { ...process.env, GH_TOKEN: token },
    })
    console.log(`[badge] Successfully updated ${filename} in Gist ${gistId}.`)
  } catch (err) {
    console.warn(`[badge] Warning: failed to update ${filename} in Gist:`, err.message)
  }
}

export const updateBadgeBranch = updateBadgeGist
