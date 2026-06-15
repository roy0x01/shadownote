package assets

import (
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"mime"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var ErrDisallowedType = errors.New("asset type not allowed")

// Block active-content assets by default; they execute as same-origin when served inline.
var activeContentExts = map[string]bool{
	".svg": true, ".svgz": true,
	".html": true, ".htm": true, ".xhtml": true, ".xml": true,
}

type Asset struct {
	Name     string    `json:"name"`
	Path     string    `json:"path"`
	URL      string    `json:"url"`
	ThumbURL string    `json:"thumb_url,omitempty"`
	MIME     string    `json:"mime"`
	Size     int64     `json:"size"`
	Modified time.Time `json:"modified"`
	Image    bool      `json:"image"`

	Used       bool     `json:"used"`
	UsageCount int      `json:"usage_count"`
	UsedBy     []string `json:"used_by,omitempty"`
}

type Store struct {
	root string
	// SVG stays opt-in because it can execute script when served inline.
	AllowSVG bool
}

func New(root string) (*Store, error) {
	if strings.TrimSpace(root) == "" {
		root = "assets"
	}
	if err := os.MkdirAll(root, 0755); err != nil {
		return nil, err
	}
	return &Store{root: root}, nil
}

func (s *Store) uploadDisallowed(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	if !activeContentExts[ext] {
		return false
	}
	if (ext == ".svg" || ext == ".svgz") && s.AllowSVG {
		return false
	}
	return true
}

func (s *Store) Root() string { return s.root }

func safeName(name string) (string, error) {
	original := strings.TrimSpace(name)
	if original == "" || strings.Contains(original, "/") || strings.Contains(original, "\\") {
		return "", fmt.Errorf("invalid asset name")
	}
	name = filepath.Base(original)
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			return r
		case r == '-', r == '_', r == '.':
			return r
		default:
			return '-'
		}
	}, name)
	name = strings.Trim(name, ".-")
	if name == "" || name == "." || name == ".." {
		return "", fmt.Errorf("invalid asset name")
	}
	return name, nil
}

func uniqueAssetPath(p string) string {
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return p
	}
	ext := filepath.Ext(p)
	base := strings.TrimSuffix(p, ext)
	for i := 1; i < 10000; i++ {
		cand := fmt.Sprintf("%s-%d%s", base, i, ext)
		if _, err := os.Stat(cand); os.IsNotExist(err) {
			return cand
		}
	}
	return p
}

func (s *Store) assetPath(name string) (string, error) {
	name, err := safeName(name)
	if err != nil {
		return "", err
	}
	root, err := filepath.Abs(s.root)
	if err != nil {
		return "", err
	}
	p := filepath.Join(root, name)
	if !strings.HasPrefix(p, root+string(os.PathSeparator)) && p != root {
		return "", fmt.Errorf("asset escapes root")
	}
	return p, nil
}

func assetMIME(name string) string {
	mt := mime.TypeByExtension(strings.ToLower(filepath.Ext(name)))
	if mt == "" {
		mt = "application/octet-stream"
	}
	return mt
}

// Only formats decoded by the standard library get generated thumbnails.
func thumbSupported(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".jpg", ".jpeg", ".png", ".gif":
		return true
	default:
		return false
	}
}

func makeAsset(root, p string, info os.FileInfo) Asset {
	rel, _ := filepath.Rel(root, p)
	rel = filepath.ToSlash(rel)
	mt := assetMIME(rel)
	isImage := strings.HasPrefix(mt, "image/")
	return Asset{Name: filepath.Base(rel), Path: rel, URL: "/assets/" + url.PathEscape(rel), MIME: mt, Size: info.Size(), Modified: info.ModTime(), Image: isImage}
}

// Avoid broken thumbnail URLs for image formats that cannot be thumbnailed.
func (s *Store) thumbURLIfExists(rel string) string {
	name, err := safeName(rel)
	if err != nil {
		return ""
	}
	thumb := filepath.Join(s.root, ".thumbs", name+".png")
	if info, err := os.Stat(thumb); err == nil && !info.IsDir() {
		return "/assets/.thumbs/" + url.PathEscape(rel) + ".png"
	}
	return ""
}

func (s *Store) decorateThumb(a *Asset) {
	if !a.Image {
		return
	}
	if thumbSupported(a.Name) {
		_ = s.ensureThumbnail(a.Path)
	}
	a.ThumbURL = s.thumbURLIfExists(a.Path)
}

func (s *Store) List() ([]Asset, error) {
	root, err := filepath.Abs(s.root)
	if err != nil {
		return nil, err
	}
	var out []Asset
	err = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".thumbs" {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		a := makeAsset(root, p, info)
		s.decorateThumb(&a)
		out = append(out, a)
		return nil
	})
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Modified.After(out[j].Modified) })
	return out, nil
}

func (s *Store) Save(name string, r io.Reader) (Asset, error) {
	if s.uploadDisallowed(name) {
		return Asset{}, fmt.Errorf("%w: %q can execute when served inline and is blocked", ErrDisallowedType, filepath.Ext(name))
	}
	p, err := s.assetPath(name)
	if err != nil {
		return Asset{}, err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return Asset{}, err
	}
	p = uniqueAssetPath(p)
	f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return Asset{}, err
	}
	if _, err := io.Copy(f, r); err != nil {
		f.Close()
		return Asset{}, err
	}
	if err := f.Close(); err != nil {
		return Asset{}, err
	}
	info, err := os.Stat(p)
	if err != nil {
		return Asset{}, err
	}
	root, _ := filepath.Abs(s.root)
	a := makeAsset(root, p, info)
	s.decorateThumb(&a)
	return a, nil
}

func (s *Store) Delete(name string) error {
	p, err := s.assetPath(name)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil {
		return err
	}
	return nil
}

func (s *Store) ensureThumbnail(rel string) error {
	name, err := safeName(rel)
	if err != nil {
		return err
	}
	src := filepath.Join(s.root, name)
	thumb := filepath.Join(s.root, ".thumbs", name+".png")
	if info, err := os.Stat(thumb); err == nil {
		if srcInfo, srcErr := os.Stat(src); srcErr == nil && !srcInfo.ModTime().After(info.ModTime()) {
			return nil
		}
	}
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	var img image.Image
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".jpg", ".jpeg":
		img, err = jpeg.Decode(f)
	case ".png":
		img, err = png.Decode(f)
	case ".gif":
		img, err = gif.Decode(f)
	default:
		return nil
	}
	if err != nil {
		return nil
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return nil
	}
	max := 320
	tw, th := w, h
	if w > h && w > max {
		tw = max
		th = h * max / w
	} else if h >= w && h > max {
		th = max
		tw = w * max / h
	}
	if tw < 1 {
		tw = 1
	}
	if th < 1 {
		th = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, tw, th))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{color.RGBA{3, 8, 6, 255}}, image.Point{}, draw.Src)
	for y := 0; y < th; y++ {
		for x := 0; x < tw; x++ {
			dst.Set(x, y, img.At(b.Min.X+x*w/tw, b.Min.Y+y*h/th))
		}
	}
	if err := os.MkdirAll(filepath.Dir(thumb), 0755); err != nil {
		return err
	}
	out, err := os.OpenFile(thumb, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer out.Close()
	return png.Encode(out, dst)
}

func (s *Store) DataURI(rel string, maxBytes int64) (string, bool) {
	name, err := safeName(rel)
	if err != nil {
		return "", false
	}
	p := filepath.Join(s.root, name)
	info, err := os.Stat(p)
	if err != nil || info.IsDir() {
		return "", false
	}
	if maxBytes > 0 && info.Size() > maxBytes {
		return "", false
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return "", false
	}
	return "data:" + assetMIME(name) + ";base64," + base64.StdEncoding.EncodeToString(data), true
}

func CopyDir(src, dst string) error {
	_, _, err := CopyDirFiltered(src, dst, nil)
	return err
}

func CopyDirFiltered(src, dst string, keep map[string]bool) (copied, skipped int, err error) {
	if _, statErr := os.Stat(src); os.IsNotExist(statErr) {
		return 0, 0, nil
	}
	err = filepath.WalkDir(src, func(p string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, relErr := filepath.Rel(src, p)
		if relErr != nil {
			return relErr
		}
		if rel == "." {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if keep != nil && !keep[filepath.ToSlash(rel)] {
			skipped++
			return nil
		}
		target := filepath.Join(dst, rel)
		in, openErr := os.Open(p)
		if openErr != nil {
			return openErr
		}
		defer in.Close()
		if mkErr := os.MkdirAll(filepath.Dir(target), 0755); mkErr != nil {
			return mkErr
		}
		out, createErr := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		if createErr != nil {
			return createErr
		}
		if _, copyErr := io.Copy(out, in); copyErr != nil {
			out.Close()
			return copyErr
		}
		copied++
		return out.Close()
	})
	return copied, skipped, err
}
