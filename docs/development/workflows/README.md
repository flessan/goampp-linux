# Proposed verification workflows

The `.yml` files **in this directory are templates**, not active GitHub Actions
workflows. GitHub only discovers workflows in `.github/workflows/`. The current
CI and release workflows remain unchanged. Nothing here grants permissions,
creates a release, or proves a build or visual comparison has passed.

| Template | Purpose |
| --- | --- |
| `windows-build.yml` | Build and upload a native Windows executable. It is a development build, not the signed/installer release build. |
| `linux-build.yml` | Build the Linux binary and generate the existing AppImage, deb, rpm and portable archive packages. |
| `cross-platform-tests.yml` | Check tracked Go formatting, vet and tests on both OSes, plus Linux race tests. |
| `ui-screenshots.yml` | Capture off-screen Windows and Linux images, upload each set, then upload side-by-side and pixel-difference images. See limitations below. |

## Prerequisites and installation

Use the Go version in `go.mod`, Git, CGO and a native compiler. Windows needs
MinGW-w64 `gcc` on PATH, and PowerShell 7. The optional
`scripts/ci/install-windows-deps.ps1` installs MinGW via Chocolatey if missing;
it requires administrative install privileges and network access. Linux needs the Fyne/GLFW, OpenGL,
X11 and Wayland development libraries; on Debian/Ubuntu,
`scripts/ci/install-linux-deps.sh` installs the same packages used by the
existing CI. Its optional `package` argument also installs ImageMagick,
linuxdeploy prerequisites and nfpm; add `$(go env GOPATH)/bin` to PATH. Package
creation downloads linuxdeploy from GitHub unless `LINUXDEPLOY` points to an
existing executable. It creates and removes staging directories inside `dist/`.
Other distributions must install equivalent dependencies manually. Python 3 and
Pillow (`python -m pip install 'Pillow>=11,<13'`) are needed for comparison.

When a maintainer with permission to modify GitHub Actions workflows has reviewed
the templates, copy the desired files into `.github/workflows/` on a branch and
commit them there. For example, from a Unix shell:

```sh
cp docs/development/workflows/ui-screenshots.yml .github/workflows/ui-screenshots.yml
```

A token or GitHub App with workflow-write permission is required to push changes
under `.github/workflows/`. The templates require repository contents read
access, normal Actions and artifact permissions, and runner/network access to Go
modules, actions, apt packages and (for packaging) linuxdeploy. Review runner
availability and action versions before installation. These templates can run
alongside the current CI and will duplicate some checks if all are enabled.

## Run locally

From the repository root:

```sh
scripts/ci/linux.sh format   # all tracked Go files; fails if any need gofmt
scripts/ci/linux.sh test     # vet and unit/integration tests
scripts/ci/linux.sh race      # Linux process lifecycle race tests
scripts/ci/linux.sh build
VERSION=dev scripts/ci/linux.sh package  # build first; packaging deps required
scripts/visual-tests/capture-linux.sh
```

On Windows PowerShell:

```powershell
./scripts/ci/windows.ps1 -Mode format
./scripts/ci/windows.ps1 -Mode test
./scripts/ci/windows.ps1 -Mode build
./scripts/visual-tests/capture-windows.ps1
```

By default the format check covers all tracked Go files, including pre-existing
formatting debt. To check only files changed since a locally available base
commit (plus untracked Go files), set `GOAMPP_FORMAT_BASE` to that commit or a
ref, for example `GOAMPP_FORMAT_BASE=origin/main scripts/ci/linux.sh format` or
`$env:GOAMPP_FORMAT_BASE = 'origin/main'` before the PowerShell command. The
cross-platform template uses the PR base SHA and full checkout history; a manual
workflow run defaults to the full repository check. The native Windows build is not the
installer or release workflow. The test job includes the Windows build smoke
test. Both capture scripts accept an optional output directory and fail if an
expected image is missing. They delete only the expected capture filenames
before running to avoid mistaking stale files for new results.

## Review screenshots

Download `ui-windows`, `ui-linux` and `ui-comparison` from the same template run.
The comparison artifact places Windows on the left and Linux on the right, with
separate absolute pixel-difference images. Locally, copy both sets to one machine
and run:

```sh
python3 scripts/visual-tests/compare.py snapshots/windows snapshots/linux snapshots/comparison
```

The script compares matching PNG filenames only, rejects dimension mismatches,
and reports unmatched files and a mean pixel difference. It does **not** assert
parity or account for font rendering, OS-specific data or differing UI state.
The Fyne test driver renders off-screen at a logical 1080 × 680; it is not a
native desktop screenshot. Windows currently captures all six pages; Linux
captures only Config Editor and Virtual Hosts with placeholder sidebar entries.
Their test data is not yet equivalent. Native dialogs, menus, DPI scaling,
fonts and service interactions still require manual, equivalent-state desktop
captures and review. See [the regression capture notes](../../ui-regression.md).
