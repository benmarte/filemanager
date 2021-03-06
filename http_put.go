package filemanager

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hacdias/filemanager/frontmatter"
)

// put is used to update a file that was edited
func put(ctx *requestContext, w http.ResponseWriter, r *http.Request) (err error) {
	var (
		data      = map[string]interface{}{}
		file      []byte
		kind      string
		rawBuffer = new(bytes.Buffer)
	)

	kind = r.Header.Get("kind")
	rawBuffer.ReadFrom(r.Body)

	if kind != "" {
		err = json.Unmarshal(rawBuffer.Bytes(), &data)

		if err != nil {
			return
		}
	}

	switch kind {
	case "frontmatter-only":
		if file, err = parseFrontMatterOnlyFile(data, r.URL.Path); err != nil {
			return
		}
	case "content-only":
		mainContent := data["content"].(string)
		mainContent = strings.TrimSpace(mainContent)
		file = []byte(mainContent)
	case "complete":
		var mark rune

		if v := r.Header.Get("Rune"); v != "" {
			var n int
			n, err = strconv.Atoi(v)
			if err != nil {
				return err
			}

			mark = rune(n)
		}

		if file, err = parseCompleteFile(data, r.URL.Path, mark); err != nil {
			return
		}
	default:
		file = rawBuffer.Bytes()
	}

	// Overwrite the request Body
	r.Body = ioutil.NopCloser(bytes.NewReader(file))
	return
}

// parseFrontMatterOnlyFile parses a frontmatter only file
func parseFrontMatterOnlyFile(data interface{}, filename string) ([]byte, error) {
	frontmatter := strings.TrimPrefix(filepath.Ext(filename), ".")
	f, err := parseFrontMatter(data, frontmatter)
	fString := string(f)

	// If it's toml or yaml, strip frontmatter identifier
	if frontmatter == "toml" {
		fString = strings.TrimSuffix(fString, "+++\n")
		fString = strings.TrimPrefix(fString, "+++\n")
	}

	if frontmatter == "yaml" {
		fString = strings.TrimSuffix(fString, "---\n")
		fString = strings.TrimPrefix(fString, "---\n")
	}

	f = []byte(fString)
	return f, err
}

// parseFrontMatter is the frontmatter parser
func parseFrontMatter(data interface{}, front string) ([]byte, error) {
	var mark rune

	switch front {
	case "toml":
		mark = '+'
	case "json":
		mark = '{'
	case "yaml":
		mark = '-'
	default:
		return nil, errors.New("Unsupported Format provided")
	}

	return frontmatter.Marshal(data, mark)
}

// parseCompleteFile parses a complete file
func parseCompleteFile(data map[string]interface{}, filename string, mark rune) ([]byte, error) {
	mainContent := ""

	if _, ok := data["content"]; ok {
		// The main content of the file
		mainContent = data["content"].(string)
		mainContent = "\n\n" + strings.TrimSpace(mainContent) + "\n"

		// Removes the main content from the rest of the frontmatter
		delete(data, "content")
	}

	if _, ok := data["date"]; ok {
		data["date"] = data["date"].(string) + ":00"
	}

	front, err := frontmatter.Marshal(data, mark)
	if err != nil {
		return []byte{}, err
	}

	front = frontmatter.AppendRune(front, mark)

	// Generates the final file
	f := new(bytes.Buffer)
	f.Write(front)
	f.Write([]byte(mainContent))
	return f.Bytes(), nil
}
