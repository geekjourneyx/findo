package skillcontent

import (
	"embed"
	"io/fs"
)

//go:embed skills/*
var embeddedSkills embed.FS

func EmbeddedSkills() (fs.FS, error) {
	return fs.Sub(embeddedSkills, "skills")
}
