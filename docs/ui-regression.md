# UI regression captures

Both captures use Fyne's off-screen test driver and a logical window size of
1080 × 680; these are not native desktop captures and do not verify system
font rendering, DPI scaling, native dialogs, tray integration or service
operations. The Windows test captures all pages; the Linux test currently
captures the Config Editor and Virtual Hosts builders only. The placeholder
pages in the Linux snapshot test exist solely to give the sidebar the normal
six entries. The test data and platforms' available actions differ, so these
captures are not yet an equivalent-state parity comparison.

On Windows, from the repository root in PowerShell:

```powershell
New-Item -ItemType Directory -Force snapshots/windows | Out-Null
$env:GOAMPP_SNAPSHOT = (Resolve-Path snapshots/windows).Path
go test ./internal/gui -run '^TestGUISnapshots$' -count=1
```

On Linux, from the repository root (with Fyne build dependencies installed):

```sh
mkdir -p snapshots/linux
GOAMPP_SNAPSHOT="$PWD/snapshots/linux" go test ./internal/linuxapp -run '^TestLinuxPageSnapshots$' -count=1
```

Compare the identically named Light and Dark images side by side. To complete
visual regression coverage, add deterministic Linux fixtures for all pages,
normalize corresponding page state, capture native UI and dialogs on both OSes,
and retain/review the generated images as CI artifacts. The current PR
connection cannot modify workflow files, so automated upload is not enabled.
