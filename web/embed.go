// Package web embeds the built admin UI. Run `pnpm build` (or `pnpm --filter web build`) first so dist exists.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

func Dist() fs.FS {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err)
	}
	return sub
}
