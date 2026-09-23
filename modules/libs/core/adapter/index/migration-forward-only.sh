#!/usr/bin/env bash
#
# A migration already in main has the contents it had there.
#
# A numbered file a build has run is a step somebody's index was brought
# through. Rewriting or deleting it moves a database this build can no longer
# reach: the diff reads as harmless, the schema is right on the machine that ran
# it, the suite stays green, and somebody else's index is broken. The schema
# goes forward by another number.
#
#   migration-forward-only.sh <base-commit>   what the branch does to the folder
#   migration-forward-only.sh --self-test     what the rule refuses, against a
#                                             repository built to be refused
#
# Exit 1 is a migration changed or removed, exit 2 is the rule having compared
# nothing.

set -euo pipefail

folder=modules/libs/core/adapter/index/migration

# The one rewrite that is not a fault: eleven files collapsed into one, once,
# against a sequence no released build had run. It is recorded as the tree the
# base carried and the file the collapse produced, so it excuses that state and
# no other — no base cut after this reaches main hashes to collapsed, and an
# edit to the file that survived is refused here as anywhere.
collapsed=e4bd40091e15a24a904179dbe17c5f830a785325
kept=0001_index.sql
produced=5fdfb401c678de79cb5d32fab5050df2b26780cf

# rule reads the folder as the base carries it against the folder as it stands.
# The comparison is with the commit the branch would merge into, so what another
# branch put on main in the meantime is not read as this branch's work.
rule() {
	local base=$1 at carried here touched

	if ! at=$(git merge-base "$base" HEAD 2>/dev/null); then
		echo "no commit is shared with $base: nothing was compared" >&2
		return 2
	fi

	carried=$(git ls-tree -r --name-only "$at" -- "$folder")
	if [ -z "$carried" ]; then
		echo "$at carries no migration under $folder/: nothing was compared" >&2
		return 2
	fi

	here=$(find "$folder" -type f -name '*.sql' 2>/dev/null || true)
	if [ -z "$here" ]; then
		echo "$folder/ holds no migration: nothing was compared" >&2
		return 2
	fi

	# The ten files the collapse took away are not read again; the one it left
	# is compared against what it produced rather than against the base.
	if [ "$(git rev-parse "$at:$folder")" = "$collapsed" ]; then
		if [ ! -f "$folder/$kept" ] ||
			[ "$(git hash-object --path "$folder/$kept" "$folder/$kept")" != "$produced" ]; then
			echo "M	$folder/$kept" >&2
			echo "the migration the collapse left is not what it produced: add a new numbered file instead" >&2
			return 1
		fi
		echo "$at carries the collapsed sequence, and $kept is what the collapse produced"
		return 0
	fi

	# Renames are off, so a migration moved to another name is the removal it
	# is. Only A is left through.
	touched=$(git diff --no-renames --name-status "$at" -- "$folder" | grep -v '^A' || true)
	if [ -n "$touched" ]; then
		echo "$touched" >&2
		echo "a migration $at already carries was changed or removed: add a new numbered file instead" >&2
		return 1
	fi
	return 0
}

# What the rule refuses, and what it lets through. A repository is built for
# each case, because a rule nothing has been seen to refuse is an assumption.
self_test() {
	local failed=0
	# The trap outlives the call, so the folder it names is not a local, and it
	# hands back the status it was reached with rather than the removal's.
	room=$(mktemp -d)
	trap 'reached=$?; rm -rf "$room"; exit $reached' EXIT

	# expect runs one fabricated branch: a name, the exit status wanted, a word
	# the report has to carry, and the commands that build the branch.
	expect() {
		local name=$1 want=$2 word=$3 build=$4
		local where="$room/$name" got=0 said

		mkdir -p "$where/$folder"
		(
			cd "$where"
			git init -q -b main .
			git config user.email check@numen.invalid
			git config user.name check
			printf 'CREATE TABLE a (id INTEGER);\n' >"$folder/0001_a.sql"
			printf 'CREATE TABLE b (id INTEGER);\n' >"$folder/0002_b.sql"
			git add -- "$folder"
			git commit -qm base
			git switch -qc branch
			eval "$build"
		)

		said=$(cd "$where" && rule "$(git -C "$where" rev-parse main)" 2>&1) || got=$?
		if [ "$got" != "$want" ]; then
			echo "$name: the rule answered $got, wanted $want" >&2
			echo "$said" >&2
			failed=1
		elif [ -n "$word" ] && [[ $said != *"$word"* ]]; then
			echo "$name: the report does not name $word" >&2
			echo "$said" >&2
			failed=1
		fi
	}

	expect added 0 '' '
		printf "CREATE TABLE c (id INTEGER);\n" > "$folder/0003_c.sql"
		git add -- "$folder"
		git commit -qm add'
	expect changed 1 0001_a.sql '
		printf "ALTER TABLE a ADD COLUMN n TEXT;\n" >> "$folder/0001_a.sql"
		git commit -qam change'
	expect removed 1 0002_b.sql '
		git rm -q -- "$folder/0002_b.sql"
		git commit -qm remove'
	expect emptied 2 "nothing was compared" '
		git rm -q -- "$folder"/*.sql
		git commit -qm empty'
	expect baseless 2 "nothing was compared" 'git checkout -q --orphan elsewhere'

	if [ "$failed" != 0 ]; then
		echo "the rule does not refuse what it is meant to refuse" >&2
		return 1
	fi
	echo "the rule refuses a changed migration, a removed one, and a folder it read nothing from"
}

case ${1:-} in
--self-test)
	self_test
	;;
"")
	echo "usage: ${0##*/} <base-commit> | --self-test" >&2
	exit 2
	;;
*)
	cd "$(git rev-parse --show-toplevel)"
	rule "$1"
	;;
esac
