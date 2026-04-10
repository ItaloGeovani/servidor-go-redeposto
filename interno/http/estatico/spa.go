package estatico

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// PossiveisPastasPainel percorridas em ordem; usa a primeira em que existir index.html (cwd do processo).
var PossiveisPastasPainel = []string{
	"assets",
	"./assets",
	"painel-web-dist",
	"./painel-web-dist",
	"../painel-web-react/dist",
	"./../painel-web-react/dist",
	"dist",
	"./dist",
}

// EncontrarRaizPainel retorna o diretorio absoluto com index.html do build Vite, ou string vazia.
func EncontrarRaizPainel() string {
	for _, dir := range PossiveisPastasPainel {
		idx := filepath.Join(dir, "index.html")
		fi, err := os.Stat(idx)
		if err != nil || fi.IsDir() {
			continue
		}
		abs, err := filepath.Abs(dir)
		if err != nil {
			return dir
		}
		return abs
	}
	return ""
}

// ComSPAFallback encerra GET/HEAD nao-API com arquivos estaticos e index.html (rotas do React).
// Rotas /saude e /v1/... seguem sempre para o handler da API.
func ComSPAFallback(api http.Handler, staticRoot string) http.Handler {
	if staticRoot == "" {
		return api
	}
	staticRoot = filepath.Clean(staticRoot)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/saude" || strings.HasPrefix(path, "/v1/") {
			api.ServeHTTP(w, r)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			api.ServeHTTP(w, r)
			return
		}
		if path == "/" {
			http.ServeFile(w, r, filepath.Join(staticRoot, "index.html"))
			return
		}
		rel := strings.TrimPrefix(path, "/")
		if rel == "" || strings.Contains(rel, "..") {
			http.Error(w, "caminho invalido", http.StatusBadRequest)
			return
		}
		full := filepath.Join(staticRoot, filepath.FromSlash(rel))
		fullClean := filepath.Clean(full)
		rootClean := filepath.Clean(staticRoot)
		relPath, err := filepath.Rel(rootClean, fullClean)
		if err != nil || strings.HasPrefix(relPath, "..") {
			http.Error(w, "acesso negado", http.StatusForbidden)
			return
		}
		if fi, err := os.Stat(fullClean); err == nil && !fi.IsDir() {
			http.ServeFile(w, r, fullClean)
			return
		}
		if strings.Contains(path, ".") {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, filepath.Join(staticRoot, "index.html"))
	})
}
