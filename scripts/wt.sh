#!/bin/sh
#
# One worktree per task. The main checkout stays on `main` and is used only
# for sync, review, merge and push; every branch starts from a fresh
# origin/main rather than a possibly stale local HEAD.
#
#	scripts/wt.sh new <slug> [feature|bugfix]
#	scripts/wt.sh list
#	scripts/wt.sh path <slug>
#	scripts/wt.sh done <slug> [--delete-branch]
#
# Worktrees are created beside the main checkout in `xinkgo-wt/`.
# XINKGO_WT_ROOT overrides that directory (absolute path).
#
set -eu

script_dir=$(CDPATH= cd "$(dirname "$0")" && pwd)
main_wt=$(git -C "$script_dir" worktree list --porcelain |
	awk '/^worktree /{print substr($0, 10); exit}')
[ -n "$main_wt" ] || {
	echo "wt.sh: cannot find the main worktree" >&2
	exit 1
}
wt_root=${XINKGO_WT_ROOT:-$(dirname "$main_wt")/xinkgo-wt}
case "$wt_root" in
/*) ;;
*)
	echo "wt.sh: XINKGO_WT_ROOT must be an absolute path: $wt_root" >&2
	exit 1
	;;
esac

git_main() { git -C "$main_wt" "$@"; }

# `git worktree` records the physical path; a root reached through a symlink
# (macOS /var -> /private/var) has to be resolved before paths are compared.
resolve_dir() { (CDPATH= cd "$1" && pwd -P); }

usage() {
	cat <<'EOF'
Usage: scripts/wt.sh <command>

  new <slug> [feature|bugfix]     Create <root>/<slug> on <type>/<slug>,
                                  starting from origin/main (default: feature)
  list                            Show all worktrees
  path <slug>                     Print the absolute path of a worktree,
                                  for use in `wt=$(scripts/wt.sh path <slug>)`
  done <slug> [--delete-branch]   Remove a clean worktree. --delete-branch
                                  also deletes the branch (only if merged)

XINKGO_WT_ROOT sets the worktree directory (default: beside the main
checkout, in `xinkgo-wt/`).
EOF
}

check_slug() {
	case "$1" in
	'' | *[!a-z0-9-]* | -* | *-)
		echo "wt.sh: bad slug '$1' (lowercase letters, digits, hyphens)" >&2
		exit 1
		;;
	esac
}

cmd_new() {
	[ "$#" -ge 1 ] && [ "$#" -le 2 ] || { usage >&2; exit 1; }
	wt_slug=$1
	wt_type=${2:-feature}
	check_slug "$wt_slug"
	case "$wt_type" in
	feature | bugfix) ;;
	*)
		echo "wt.sh: type must be feature or bugfix, got '$wt_type'" >&2
		exit 1
		;;
	esac
	wt_branch="$wt_type/$wt_slug"

	if git_main show-ref --verify --quiet "refs/heads/$wt_branch"; then
		echo "wt.sh: branch '$wt_branch' already exists" >&2
		exit 1
	fi
	mkdir -p "$wt_root"
	wt_root=$(resolve_dir "$wt_root")
	wt_path="$wt_root/$wt_slug"
	[ ! -e "$wt_path" ] || {
		echo "wt.sh: '$wt_path' already exists" >&2
		exit 1
	}

	git_main fetch origin
	git_main rev-parse --verify --quiet origin/main >/dev/null || {
		echo "wt.sh: origin/main not found" >&2
		exit 1
	}
	git_main worktree add -b "$wt_branch" "$wt_path" origin/main

	printf '\nWorktree: %s\nBranch:   %s\n\n' "$wt_path" "$wt_branch"
	printf '  codex -C "%s"\n' "$wt_path"
	printf '  wt=$(scripts/wt.sh path %s)\n' "$wt_slug"
	printf '  scripts/check.sh               # in the worktree, before committing\n'
	printf '  scripts/wt.sh done %s         # back in the main checkout\n' "$wt_slug"

	wt_cfg="${CODEX_HOME:-$HOME/.codex}/config.toml"
	if [ -f "$wt_cfg" ] && grep -Fq 'writable_roots' "$wt_cfg" &&
		! grep -Fq "$wt_root" "$wt_cfg"; then
		printf '\nNote: add "%s" to writable_roots in %s (or use --add-dir).\n' \
			"$wt_root" "$wt_cfg"
	fi
}

cmd_list() {
	[ "$#" -eq 0 ] || { usage >&2; exit 1; }
	git_main worktree list
}

cmd_path() {
	[ "$#" -eq 1 ] || { usage >&2; exit 1; }
	wt_slug=$1
	check_slug "$wt_slug"
	[ -d "$wt_root" ] || {
		echo "wt.sh: no worktree directory '$wt_root'" >&2
		exit 1
	}
	wt_root=$(resolve_dir "$wt_root")
	wt_path="$wt_root/$wt_slug"
	[ -d "$wt_path" ] || {
		echo "wt.sh: no worktree at '$wt_path'" >&2
		exit 1
	}
	wt_toplevel=$(git -C "$wt_path" rev-parse --show-toplevel 2>/dev/null || true)
	[ "$wt_toplevel" = "$wt_path" ] || {
		echo "wt.sh: '$wt_path' is not a worktree of this repo" >&2
		exit 1
	}
	printf '%s\n' "$wt_path"
}

cmd_done() {
	[ "$#" -ge 1 ] && [ "$#" -le 2 ] || { usage >&2; exit 1; }
	wt_slug=$1
	check_slug "$wt_slug"
	wt_delete=0
	if [ "$#" -eq 2 ]; then
		case "$2" in
		--delete-branch) wt_delete=1 ;;
		*)
			echo "wt.sh: unknown option '$2'" >&2
			exit 1
			;;
		esac
	fi
	[ -d "$wt_root" ] || {
		echo "wt.sh: no worktree directory '$wt_root'" >&2
		exit 1
	}
	wt_root=$(resolve_dir "$wt_root")
	wt_path="$wt_root/$wt_slug"

	[ -d "$wt_path" ] || {
		echo "wt.sh: no worktree at '$wt_path'" >&2
		exit 1
	}
	[ "$wt_path" != "$main_wt" ] || {
		echo "wt.sh: refusing to remove the main worktree" >&2
		exit 1
	}
	wt_toplevel=$(git -C "$wt_path" rev-parse --show-toplevel 2>/dev/null || true)
	[ "$wt_toplevel" = "$wt_path" ] || {
		echo "wt.sh: '$wt_path' is not a worktree of this repo" >&2
		exit 1
	}
	wt_status=$(git -C "$wt_path" status --porcelain)
	[ -z "$wt_status" ] || {
		echo "wt.sh: '$wt_path' has uncommitted changes:" >&2
		printf '%s\n' "$wt_status" >&2
		exit 1
	}
	wt_branch=$(git -C "$wt_path" symbolic-ref --quiet --short HEAD || true)
	if [ "$wt_delete" = 1 ] && [ -z "$wt_branch" ]; then
		echo "wt.sh: detached HEAD, no branch to delete" >&2
		exit 1
	fi
	if [ -n "$wt_branch" ]; then
		wt_upstream=$(git -C "$wt_path" rev-parse --abbrev-ref \
			--symbolic-full-name '@{u}' 2>/dev/null || true)
		if [ -z "$wt_upstream" ]; then
			printf 'Warning: %s has no upstream; push it before requesting a merge.\n' \
				"$wt_branch" >&2
		else
			wt_ahead=$(git -C "$wt_path" rev-list --count "$wt_upstream..HEAD")
			[ "$wt_ahead" = 0 ] || printf 'Warning: %s has %s unpushed commit(s).\n' \
				"$wt_branch" "$wt_ahead" >&2
		fi
	fi

	git_main worktree remove "$wt_path"
	git_main worktree prune
	printf 'Removed worktree %s\n' "$wt_path"
	if [ "$wt_delete" = 1 ]; then
		git_main branch -d "$wt_branch"
	else
		printf 'Branch %s kept; delete it from the main checkout after the merge.\n' \
			"${wt_branch:-<detached HEAD>}"
	fi
}

case "${1:-}" in
new) shift; cmd_new "$@" ;;
list) shift; cmd_list "$@" ;;
path) shift; cmd_path "$@" ;;
done) shift; cmd_done "$@" ;;
*) usage >&2; exit 1 ;;
esac
