# Releasing

Five steps, in order. Nothing here is optional, and step 4 is its own commit
for a reason: a release must be able to name the exact commit it shipped.

## 1 · Branch

Branch off a freshly-fetched `master`, named for what it does.

```sh
git fetch origin
git checkout -b feat/choose-disk origin/master
```

`feat/` for a feature, `fix/` for a defect, `docs/` for prose. The branch name
ends up in the squash-merge subject, so make it readable.

## 2 · PR

Open it before you are finished, so CI runs on the work in progress.

```sh
git push -u origin feat/choose-disk
gh pr create --fill --base master
```

`--fill` takes the title and body from your commits. Edit the body to say what
changed and what you verified; the diff already says how.

Wait for the checks. `Build` must be green — it runs `gofmt`, `mygo vet`, the
tests, and a build of both Windows architectures.

## 3 · Squash merge

```sh
gh pr merge <n> --squash --delete-branch
```

One commit per change on `master`, whatever the branch looked like. The branch
goes away with it.

## 4 · Bump

Its own commit, so the version bump is never mixed into a feature diff.

```sh
bun run version:bump <next>
```

This repo has **no `package.json`** — it is a Go project — so `bun run
version:bump` does not exist here. The equivalent, and the one to use:

```sh
go run ./internal/tools/versionbump 0.2.0
git add mygo.json && git commit -m "Bump to 0.2.0" && git push
```

`mygo.json` is the one place the version lives: the build names the installer
from it, and `release.yml` checks the tag against it. The tool rewrites that
single field in place and refuses anything that is not a version, so a bump is
always a one-line diff.

If you would rather have `bun run version:bump` as the documented interface,
add a `package.json` with that script — but a Go project that needs Bun
installed to cut a release is a worse trade than one command that is four words
longer.

## 5 · Release

```sh
gh workflow run release.yml -f tag=v0.2.0
```

The workflow:

- refuses to run when the tag and `mygo.json` disagree, so a release can never
  be named something other than what it contains;
- runs `gofmt`, `mygo vet` and the tests;
- builds `windows/amd64` and `windows/arm64`;
- publishes `Sweep-setup.exe` (the amd64 installer, the link the site points
  at), plus per-architecture names for both architectures.

Then **publish the draft by hand**: open the release, read the generated notes,
and edit them into something a person would want to read. The notes are a
starting point, not the release.

```sh
gh release view v0.2.0 --web
```

Publishing a draft creates the git tag, which fires `release.yml` a second time
through its `v*` trigger. The workflow's first step checks whether the tag
already has a release and stops there, so the second run does nothing. If you
ever see two releases for one tag, that guard is what is missing.

### After publishing

Check the download the site offers actually resolves, because it is the one
link most people will use:

```sh
curl -sIL -o /dev/null -w '%{http_code}\n' \
  https://github.com/yaffalhakim1/sweep/releases/latest/download/Sweep-setup.exe
```

`200` means the alias asset is in place. A `404` means the release is missing
`Sweep-setup.exe`, and the landing page's Download button is broken.

---

## What CI enforces

| Workflow | Runs on | Does |
|---|---|---|
| `build.yml` | push to `master`, every PR | `gofmt`, `mygo vet`, tests, build both architectures |
| `release.yml` | a `v*` tag, or dispatch | the above, plus tag/version check, and publishes |
| `pages.yml` | changes under `landing/` | publishes the landing page |

## Things that have bitten before

- **Line endings.** The runner checks out with `autocrlf=true`, which turns LF
  into CRLF and makes `gofmt` flag every file. `.gitattributes` pins `eol=lf`;
  without it the formatting check fails on files nobody edited.
- **Asset names collide.** Both architectures build `Sweep.exe`, and a release
  cannot hold two assets of one name, so the workflow renames them per
  architecture.
- **The download link must be version-less.** A URL containing the version
  breaks on the next release, which is why the workflow publishes
  `Sweep-setup.exe` as a stable alias.
- **The version and the tag must agree.** A user who downloads
  `v0.2.0` and gets an installer claiming `0.1.0` cannot tell what they have.
  The workflow fails rather than publish that.
- **Publishing a draft re-triggers the release.** Publishing creates the tag,
  the tag fires `release.yml`, and the run builds a second release. The
  workflow's guard step is what stops it: it checks whether the tag already has
  a release and skips the rest of the job if it does. This happened once — the
  `v0.2.0` release briefly existed twice — which is why the guard is there.

## Not set up: signed auto-update

`mygo` can sign update archives so an installed app updates itself, and the
sibling `ports` project uses it. This project does not, deliberately: it needs
a key pair and a private key stored as a secret, and a lost key strands every
installed copy. A user who wants a newer version downloads the installer,
which installs over the old one.

To add it: `go tool mygo keygen`, add an `updates` block to `mygo.json`, store
the private key as `MYGO_UPDATER_PRIVATE_KEY`, pass it to the build in
`release.yml`, and call `mygo.Use(native.Plugin)` from the updater plugin in
`main.go`. The tradeoff is above — decide it on purpose.
