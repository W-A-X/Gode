package engine

import (
	"os"

	"github.com/gogpu/ui/plugin"
)

// cjkFontFamily is the family name under which a system CJK font is
// registered in the process-global font registry. The EditorView uses it to
// render CJK glyphs: the embedded Inter font has no CJK coverage, so without
// a fallback Chinese/Japanese/Korean text renders as tofu boxes.
const cjkFontFamily = "GodeCJK"

// systemCJKFontCandidates lists platform font files that contain CJK glyphs,
// in order of preference. TTF files are preferred (no collection indexing);
// TTC collections fall back to their first face. The first readable and
// parseable file wins.
var systemCJKFontCandidates = []string{
	// macOS
	"/System/Library/Fonts/Supplemental/Arial Unicode.ttf",
	"/System/Library/Fonts/PingFang.ttc",
	"/System/Library/Fonts/Hiragino Sans GB.ttc",
	"/System/Library/Fonts/STHeiti Medium.ttc",
	"/System/Library/Fonts/STHeiti Light.ttc",
	// Linux (Noto CJK)
	"/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc",
	"/usr/share/fonts/truetype/noto/NotoSansCJK-Regular.ttc",
	// Windows
	"C:\\Windows\\Fonts\\msyh.ttc",   // Microsoft YaHei
	"C:\\Windows\\Fonts\\simsun.ttc", // SimSun
	"C:\\Windows\\Fonts\\msyh.ttf",
}

// registerCJKFont loads the first available system CJK font into the global
// font registry. It returns the family name to use for CJK rendering, or ""
// when no usable font was found (the editor then falls back to Inter).
func registerCJKFont() string {
	ctx := plugin.NewDefaultPluginContext()
	for _, path := range systemCJKFontCandidates {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if err := ctx.Assets.LoadFont(cjkFontFamily, data); err == nil {
			return cjkFontFamily
		}
	}
	return ""
}
