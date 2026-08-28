# Desktop release

Desktop releases use one packaging chain locally and in GitHub Actions:

1. Next.js exports static assets for Go embedding.
2. `build-python-runtime.sh` builds a target-native, self-contained Python
   runtime and copies the engine source into it.
3. Wails builds `AlchemyFurnace.app` or `AlchemyFurnace.exe` with release
   metadata injected through Go linker flags.
4. `ci-assemble.sh` installs the runtime, signs and packages the macOS bundle,
   or builds the Windows NSIS installer.
5. The release job gates every asset through `scripts/verify-release-assets.sh`
   (5 platform binaries, `checksums.txt`, `release-manifest.json`), creates a
   draft Release, re-verifies the assets downloaded from the draft, and only
   then publishes.

## Release integrity gate (draft flow)

The `Release desktop` workflow never publishes a Release directly. The release
aggregate job runs this gate:

1. `scripts/verify-release-assets.sh dist <version> --binaries-only` — checks
   that exactly the 5 platform assets exist, are non-empty, and contain no
   duplicate architecture.
2. Generates `checksums.txt` (SHA256 of the 5 binaries) and
   `release-manifest.json` (`version`, `tag`, build metadata, and per-asset
   `size`/`sha256`) inside the dist directory.
3. Re-runs `scripts/verify-release-assets.sh dist <version>` in full mode —
   missing/empty files, duplicate architectures, checksums mismatches, unknown
   files, or a manifest `version` that differs from the tag all fail the job.
4. Creates a **draft** Release (never `latest`) with only the whitelisted
   assets: the 5 binaries, `checksums.txt`, and `release-manifest.json`.
5. Downloads the draft assets to an empty directory
   (`gh release download`) and runs the same verifier against them, so the
   published bytes are proven identical to the verified bytes.
6. Only after the remote re-verification passes does
   `gh release edit "$TAG" --draft=false --latest` publish the Release.

Any failure at any step keeps the Release a draft for inspection — it is never
published and never marked `latest`. A failing package job skips the release
job entirely (`needs` on the matrix job).

**Do not manually publish partial assets.** Uploading a subset of platform
binaries by hand bypasses the gate and breaks the updater for the missing
architectures; emergency releases must still go through the draft integrity
gate with all 5 assets verified.

Local usage of the verifier:

```bash
scripts/verify-release-assets.sh <dist-directory> <version> [--binaries-only]
```

`<version>` may be with or without a leading `v`; it is compared against the
`version` field of `release-manifest.json` after normalization.

## Automatic release

Create and push a semantic-version tag:

```bash
git tag v0.2.0
git push origin v0.2.0
```

The `Release desktop` workflow can also be dispatched manually for an existing
tag. It is callable from another workflow through `workflow_call`, with `tag`
as its input.

## Local package

Run on the target architecture so Python wheels and native Wails libraries
match the package:

```bash
make desktop-package PLATFORM=darwin-arm64 VERSION=v0.2.0
make desktop-package PLATFORM=darwin-amd64 VERSION=v0.2.0
make desktop-package PLATFORM=windows-amd64 VERSION=v0.2.0
```

Required tools are Go, Node.js, pnpm, and Wails. Windows additionally needs
NSIS; macOS uses the system `codesign`, `hdiutil`, and `ditto` tools.

## Signing

macOS bundles are ad-hoc signed after the Python runtime is assembled. This is
important: modifying `Contents/Resources` after signing invalidates the bundle.
Set `MACOS_SIGN_IDENTITY` only on a runner where that identity and its private
key are already installed. Apple notarization and Windows Authenticode signing
remain credential-dependent deployment steps.
