# Releasing mail modules

`scripts/module-manifest.txt` is the source of truth for module ownership. The
root module and `mailses` are published modules. The `docs` and `examples`
modules are repository tooling and must not receive release tags.

## Pre-release validation

The root `v0.3.0` tag does not exist while the release is being staged, so the
public module boundary cannot yet resolve the version required by `mailses`.
Validate the staged source with temporary modfiles instead:

```bash
scripts/check-published-modules.sh v0.3.0
MAIL_LOCAL_SIBLINGS=1 MAIL_RELEASE_VERSION=v0.3.0 scripts/check-quality.sh
go test -count=1 -race ./...
(cd mailses && go test -count=1 -race ./...)
git diff --check
```

`MAIL_LOCAL_SIBLINGS=1` keeps `GOWORK=off` for every module command and gives
each published module a disposable modfile containing absolute local sibling
replacements. It never changes a committed manifest or claims that an
unpublished tag already resolves.

## Dependency-ordered release

The root tag must be published before the SES module because `mailses/v0.3.0`
imports the root module's shared HTTP and MIME implementation.

1. From the reviewed, clean quality-pass commit, tag and push root `v0.3.0`:

   ```bash
   git tag -a v0.3.0 -m "release v0.3.0"
   git push origin v0.3.0
   ```
2. Wait until this succeeds:

   ```bash
   GOWORK=off go mod download github.com/goforj/mail@v0.3.0
   ```

3. Resolve the real published dependency and validate without the workspace:

   ```bash
   (
     cd mailses
     GOWORK=off go mod tidy
     GOWORK=off go vet ./...
     GOWORK=off go test -count=1 ./...
     GOWORK=off go test -count=1 -race ./...
   )
   MAIL_RELEASE_VERSION=v0.3.0 scripts/check-quality.sh
   ```

4. Review and commit the checksum-only staging change. No root source or
   `mailses` source may change after the root tag:

   ```bash
   git diff -- mailses/go.mod mailses/go.sum
   git add mailses/go.mod mailses/go.sum
   git commit -m "chore(release): stage mailses v0.3.0 checksums"
   git diff --exit-code v0.3.0..HEAD -- . ':(exclude)mailses/go.sum'
   ```

5. Tag and publish `mailses/v0.3.0` from that checksum staging commit:

   ```bash
   git tag -a mailses/v0.3.0 -m "release mailses/v0.3.0"
   git push origin mailses/v0.3.0
   ```

6. After both tags are visible through the public proxy, fetch and test each
   exact published module from fresh consumers:

   ```bash
   scripts/check-public-release.sh v0.3.0
   ```

   This check explicitly resolves each `module@v0.3.0`; running tests from the
   checkout alone would validate local source rather than the published tag.
   It uses a fresh module cache and the public Go proxy without direct-VCS
   fallback. Set `MAIL_RELEASE_GOPROXY` to a different proxy-only endpoint when
   validating a private or mirrored release.

The two release tags intentionally point at adjacent commits: the second commit
can only record checksums that become available after publishing the root tag.
Published module files must never use a sibling `replace` directive or a
`v0.0.0` placeholder.
