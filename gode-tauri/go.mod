module github.com/gode/gode-tauri

go 1.21

require (
	github.com/gogpu/gogpu v0.0.0
	github.com/gogpu/ui v0.0.0
	gode/editor v0.0.0
)

replace (
	gode/editor => ../editor-go
)
