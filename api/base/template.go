package base

import (
	"errors"
	"html/template"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/caitunai/dotconfig/storage"
	"github.com/gin-gonic/gin/render"
	"github.com/rs/zerolog/log"
)

type Renderer interface {
	render.HTMLRender
	LoadTemplates()
}

// Render type
type Render map[string]*template.Template

var (
	_               render.HTMLRender = Render{}
	_               Renderer          = Render{}
	ErrReadLayout                     = errors.New("error to read layout file")
	ErrPageTemplate                   = errors.New("error to read page template")
)

// NewRender instance
func NewRender() Render {
	r := make(Render)
	r.LoadTemplates()
	return r
}

// Instance supply render string
func (r Render) Instance(name string, data any) render.Render {
	return render.HTML{
		Template: r[name],
		Name:     name,
		Data:     data,
	}
}

func (r Render) LoadTemplates() {
	root := "views"
	layoutFolder := "layout"
	sharedFolder := "shared"
	includes, _ := fs.Glob(storage.FS, root+"/**/*.html")

	// Generate our templates map from our layout/ and shared/ directories
	layouts := make([]string, 0)
	for _, include := range includes {
		if strings.Contains(include, layoutFolder) || strings.Contains(include, sharedFolder) {
			layouts = append(layouts, include)
		}
	}
	var err error
	var content []byte
	tpl := template.New("tpl")
	for _, file := range layouts {
		if r.isLayout(file) || r.isShared(file) {
			// 布局模版和共享组件
			content, err = r.readLayoutFile(file)
		}
		if err == nil {
			_, err = tpl.Parse(string(content))
			if err != nil {
				log.Error().Err(err).Msgf("Parse html templates failed: %s", file)
			}
		}
	}
	for _, include := range includes {
		if !strings.Contains(include, layoutFolder) && !strings.Contains(include, sharedFolder) {
			// remove .html
			pageName := strings.TrimSuffix(include, filepath.Ext(include))
			// remove .app or .tplName
			pageName = strings.TrimSuffix(pageName, filepath.Ext(pageName))
			pageName = strings.TrimPrefix(pageName, root+"/")
			pageName = strings.ReplaceAll(pageName, "/", ".")
			cloneTpl, cErr := tpl.Clone()
			if cErr == nil {
				tmpl := cloneTpl.New(pageName)
				// read page
				content, err = r.readPageTemplateFile(include, pageName)
				if err == nil {
					_, err = tmpl.Parse(string(content))
					if err != nil {
						log.Error().Err(err).Msgf("Parse html templates failed: %s", include)
					}
				}
				r[pageName] = tmpl
			} else {
				log.Error().Err(cErr).Str("pageName", pageName).Msgf("clone html template failed: %s", include)
			}
		}
	}
}

func (r Render) isLayout(file string) bool {
	return strings.Contains(file, "/layout/")
}

func (r Render) isShared(file string) bool {
	return strings.Contains(file, "/shared/")
}

func (r Render) getFileName(fileName string) string {
	return strings.TrimSuffix(filepath.Base(fileName), filepath.Ext(fileName))
}

func (r Render) getTemplateName(fileName string) string {
	if strings.Contains(fileName, ".") {
		return strings.Trim(filepath.Ext(fileName), ".")
	}
	return ""
}

func (r Render) readLayoutFile(file string) ([]byte, error) {
	content, err := storage.FS.ReadFile(file)
	if err == nil {
		tplName := r.getFileName(file)
		prefix := []byte("{{ define \"" + tplName + "\" }}")
		suffix := []byte("{{ end }}")
		content = append(prefix, content...)
		content = append(content, suffix...)
	} else {
		err = errors.Join(err, ErrReadLayout)
	}
	return content, err
}

func (r Render) readPageTemplateFile(file, pageName string) ([]byte, error) {
	content, err := storage.FS.ReadFile(file)
	if err == nil {
		tplName := r.getFileName(file)
		tmpName := r.getTemplateName(tplName)
		if tmpName == "" {
			prefix := []byte("{{ define \"" + pageName + "\" }}")
			suffix := []byte("{{ end }}")
			content = append(prefix, content...)
			content = append(content, suffix...)
			return content, nil
		}
		prefix := []byte("{{ define \"" + pageName + "\" }}{{ template \"" + tmpName + "\" .}}{{ end }}")
		content = append(prefix, content...)
	} else {
		err = errors.Join(err, ErrPageTemplate)
	}
	return content, err
}
