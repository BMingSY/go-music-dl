"""Subset the icon fonts and package the source PNG without enlarging it.

Requires fonttools==4.68.0. Run from the repository root.
"""

import hashlib
import io
from pathlib import Path
import re
import struct
import urllib.request

from fontTools import subset
from fontTools.ttLib import TTFont


root = Path(__file__).resolve().parent.parent
assets = root / "assets"
codepoints = {int(x, 16) for x in re.findall(r"\\u([0-9a-fA-F]{4})", (root / "assets.go").read_text())}
sources = [
    ("fa-solid-900.ttf", "af19d135d3a935b3ebfbd80320716ffe1202052c5f68dc2c5f1abc57005ac605", "musicdl-icons.ttf", "MusicDL Icons"),
    ("fa-regular-400.ttf", "557e6966ad8dba7352ac6d350c1546b685b13d68db26f09cba45cadd660d45db", "musicdl-outline.ttf", "MusicDL Outline"),
]
for source, digest, output, family in sources:
    url = f"https://raw.githubusercontent.com/FortAwesome/Font-Awesome/6.7.2/webfonts/{source}"
    data = urllib.request.urlopen(url, timeout=60).read()
    assert hashlib.sha256(data).hexdigest() == digest, f"Checksum mismatch: {source}"
    font = TTFont(io.BytesIO(data), recalcTimestamp=False)
    options = subset.Options()
    options.recalc_timestamp = False
    options.name_IDs = [0, 1, 2, 3, 4, 5, 6, 13, 14, 16, 17]
    options.name_legacy = True
    options.name_languages = [0x409]
    sub = subset.Subsetter(options=options)
    sub.populate(unicodes=codepoints & font.getBestCmap().keys())
    sub.subset(font)
    # Font Awesome is a reserved font name under the OFL.
    for record in font["name"].names:
        names = {1: family, 2: "Regular", 3: family.replace(" ", ""), 4: family,
                 5: "Version 1.0", 6: family.replace(" ", ""), 16: family, 17: "Regular"}
        if record.nameID in names:
            record.string = names[record.nameID].encode(record.getEncoding())
    font.save(assets / output)
    print(output, (assets / output).stat().st_size)

# The original artwork is 256 pixels. Finder can scale it for larger icons;
# storing several enlarged copies adds no source detail.
png = (assets / "icon.png").read_bytes()
entries = b"".join(kind + struct.pack(">I", len(png) + 8) + png for kind in (b"ic08", b"ic13"))
(assets / "icon.icns").write_bytes(b"icns" + struct.pack(">I", len(entries) + 8) + entries)
