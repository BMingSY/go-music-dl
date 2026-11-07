# Build assets

The fonts in `internal/assets` contain only glyphs used in `internal/desktop/assets.go`. To regenerate them after adding an icon:

```sh
python3 -m pip install fonttools==4.68.0
python3 tools/prepare-assets.py
```

The script fetches Font Awesome 6.7.2 with SHA-256 verification, subsets and renames the fonts under the SIL OFL, and packages the existing PNG at its original resolution for macOS. It does not change the artwork.

See [bundle-ffmpeg](bundle-ffmpeg/README.md) for media builds.
