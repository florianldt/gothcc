package components

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
)

// assetVersions maps static file URLs (/static/css/app.css) to a content
// hash. A CDN like Cloudflare caches /static, so an unversioned URL keeps
// serving the old file after a deploy; asset adds ?v=<hash> so each change
// gets a new URL.
var assetVersions = map[string]string{}

// VersionAssets hashes every file under dir (relative to the working
// directory, e.g. "static"); call it once at startup. In dev, air restarts
// the server on every rebuild, which picks up new hashes.
func VersionAssets(dir string) {
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		assetVersions["/"+filepath.ToSlash(path)] = hex.EncodeToString(sum[:])[:12]
		return nil
	})
	if err != nil {
		slog.Warn("static assets not versioned", "dir", dir, "error", err)
	}
}

// asset returns the URL of a static file with its content hash, or the bare
// URL when the file was not hashed (tests, a missing file).
func asset(url string) string {
	if v, ok := assetVersions[url]; ok {
		return url + "?v=" + v
	}
	return url
}
