# Tags

Reads the tags of your notes from their frontmatter and lists them for agents, with the number of notes under each tag.

<!-- more -->

## What You Can Do

- **Tag a note**: Write the tags of a note under the key `tags` of its frontmatter, as a list (`tags: [Projects/Web, River]`) or as one tag (`tags: River`).
- **Nest tags**: A slash puts a tag under another: `Projects/Web` is under `Projects`, and a note tagged `Projects/Web` counts for both.
- **Ask an agent about your tags**: An agent lists the tags of the vault with the tool `tag_list`, the whole tree or the tags under one tag.

## How Tags Are Read

- A tag is the same tag in any case: `River`, `river` and `RIVER` are one tag. The list shows the spelling of the first note in path order.
- A leading `#` is not part of the tag, so `#River` is `River`.
- Spaces around a tag and around each level are dropped, and an empty level is left out: `Projects / Web` is `Projects/Web`.
- A number is read as its text. Any other value of `tags`, such as a map, gives no tags.
- A note counts once for a tag, however many of its tags are that tag or nested under it.

The plugin reads every note's tags the first time it is used after Numen starts, and reads a note again when its frontmatter changes. **Update tags** in the palette reads the tags of the note in front again.

## For Plugin Authors

The plugin has the permission `vault:read`. It reads the frontmatter key `tags` of the notes through `ReadFrontmatter` of the host, in pages at its first use of a vault, and keeps the tags of each note in `tags.json` in its folder of the vault's store. Its trigger on `note.frontmatterChanged` runs its own command `tags.update` over the notes whose frontmatter changed, up to 500 in one call, which reads those notes again in one `ReadFrontmatter`. The tool `tag_list` (capability `tools`) returns the tree depth first, each tag with its depth and the number of notes that have it or a tag nested under it; `under` lists one tag and the tags nested under it.
