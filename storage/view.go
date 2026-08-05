package storage

import (
	"embed"
	"encoding/json"
	"io/fs"
)

//go:embed views ui/dist static
var FS embed.FS

var (
	UI, _     = fs.Sub(FS, "ui/dist")
	Assets, _ = fs.Sub(UI, "assets")
	Static, _ = fs.Sub(FS, "static")
)

type ManifestNode struct {
	Src     string   `json:"src"`
	File    string   `json:"file"`
	CSS     []string `json:"css"`
	Imports []string `json:"imports"`
	IsEntry bool     `json:"isEntry"`
}

type Manifest map[string]*ManifestNode

func ParseManifest() Manifest {
	file, err := FS.ReadFile("ui/dist/manifest.json")
	if err != nil {
		return nil
	}
	node := make(Manifest)
	err = json.Unmarshal(file, &node)
	if err != nil {
		return nil
	}
	return node
}

func (m Manifest) GetCSSFiles(entry string) []string {
	node, ok := m[entry]
	if !ok {
		return []string{}
	}
	return node.CSS
}

func (m Manifest) GetJsFiles(entry string) []string {
	node, ok := m[entry]
	if !ok {
		return []string{}
	}
	files := make([]string, 0)
	files = append(files, node.File)
	if len(node.Imports) > 0 {
		for _, v := range node.Imports {
			inode, exist := m[v]
			if exist {
				files = append(files, inode.File)
			}
		}
	}
	return files
}
