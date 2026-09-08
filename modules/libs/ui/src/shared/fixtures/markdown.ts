/**
 * Marked-up text with one of everything in it.
 *
 * Every construct the editor draws appears here once, so a story shows the
 * whole vocabulary and a change that breaks one of them is visible.
 */

/** A picture that needs nothing from the network. */
export const PICTURE =
  'data:image/svg+xml;utf8,' +
  encodeURIComponent(
    '<svg xmlns="http://www.w3.org/2000/svg" width="240" height="80">' +
      '<rect width="240" height="80" rx="8" fill="#3f6ca8"/>' +
      '<circle cx="60" cy="40" r="18" fill="#fbfbfa"/>' +
      '<circle cx="180" cy="40" r="10" fill="#6fbd93"/>' +
      '<line x1="78" y1="40" x2="170" y2="40" stroke="#fbfbfa" stroke-width="2"/>' +
      '</svg>',
  )

export const MARKED_UP = `# Entropy

Some of this is **strong**, some *slanted*, some ~~struck out~~, and some is
\`inline code\`. A link goes to
[somewhere](https://example.invalid/vault "a title"), and
<https://example.invalid/plain> stands on its own.

## What follows from it

> A quotation, which keeps its own line
> for as long as it runs.

- a bullet
- another one
  - and one below it
- [ ] something to do
- [x] something done

1. first
2. second

---

### A table

| Word | Kind | Written |
|:-----|:----:|--------:|
| Entropy | parent | 1865 |
| Temperature | child | 1848 |
| Heat | jump | 1798 |

### Code

\`\`\`go
func main() {
	// Nothing here is measured twice.
	name := "entropy.md"
	fmt.Println(name)
}
\`\`\`

\`\`\`python
def neighbours(thing):
    return [link.to for link in thing.links]
\`\`\`

![a picture](${PICTURE})
`

/** A table on its own, for typing into. */
export const TABLE = `| Word | Kind | Written |
|:-----|:----:|--------:|
| Entropy | parent | 1865 |
| Temperature | child | 1848 |
`
