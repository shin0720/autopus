package cli

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func handleFileUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	root := getWorkspaceDir()
	if root == "" {
		root = uiProjectRoot
	}
	var uploaded []string
	for _, headers := range r.MultipartForm.File {
		for _, fh := range headers {
			name := filepath.Base(strings.ReplaceAll(fh.Filename, "\\", "/"))
			src, err := fh.Open()
			if err != nil {
				continue
			}
			dst, err := os.Create(filepath.Join(root, name))
			if err != nil {
				src.Close()
				continue
			}
			_, _ = io.Copy(dst, src)
			src.Close()
			dst.Close()
			uploaded = append(uploaded, name)
		}
	}
	if uploaded == nil {
		uploaded = []string{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"uploaded": uploaded})
}
