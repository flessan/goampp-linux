#!/usr/bin/env python3
"""Produce side-by-side and absolute pixel difference images for matching PNGs."""
import argparse
from pathlib import Path
import sys

try:
    from PIL import Image, ImageChops, ImageStat
except ImportError:
    sys.exit("Pillow is required: python -m pip install Pillow")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("windows", type=Path)
    parser.add_argument("linux", type=Path)
    parser.add_argument("output", type=Path)
    args = parser.parse_args()
    if not args.windows.is_dir() or not args.linux.is_dir():
        parser.error("both input directories must exist")
    windows = {p.name: p for p in args.windows.glob("*.png")}
    linux = {p.name: p for p in args.linux.glob("*.png")}
    common = sorted(windows.keys() & linux.keys())
    if not common:
        sys.exit("No matching PNG filenames; no comparison was performed")
    args.output.mkdir(parents=True, exist_ok=True)
    print("Only matching filenames are compared; review state and OS-specific controls manually.")
    print("Windows-only:", ", ".join(sorted(windows.keys() - linux.keys())) or "none")
    print("Linux-only:", ", ".join(sorted(linux.keys() - windows.keys())) or "none")
    failed = False
    for name in common:
        with Image.open(windows[name]) as w, Image.open(linux[name]) as l:
            w, l = w.convert("RGB"), l.convert("RGB")
            if w.size != l.size:
                print(f"{name}: size mismatch Windows={w.size} Linux={l.size}", file=sys.stderr)
                failed = True
                continue
            side = Image.new("RGB", (w.width * 2, w.height))
            side.paste(w, (0, 0))
            side.paste(l, (w.width, 0))
            diff = ImageChops.difference(w, l)
            side.save(args.output / f"{Path(name).stem}_side-by-side.png")
            diff.save(args.output / f"{Path(name).stem}_difference.png")
            mean = sum(ImageStat.Stat(diff).mean) / 3
            print(f"{name}: {w.width}x{w.height}, mean RGB difference {mean:.2f}/255")
    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main())
