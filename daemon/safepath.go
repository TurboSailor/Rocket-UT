package main

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Ограничение файловых операций каталогами пользователя.
// Перенос safe_dir/safe_file из awgd.py: realpath + проверка вхождения в корни,
// отказ на симлинки и нерегулярные файлы.

var errNotAllowed = errors.New("path not allowed")

var confExt = []string{".conf", ".txt", ".json", ".ini"}

var imgExt = []string{".png", ".jpg", ".jpeg", ".bmp", ".webp"}

func okPath(p string) bool {
	return p != "" && !strings.ContainsRune(p, 0) && len(p) <= 4096
}

func realPath(p string) (string, bool) {
	if !okPath(p) {
		return "", false
	}
	rp, err := filepath.EvalSymlinks(p)
	if err != nil {
		// Файла может не быть — работаем с очищенным абсолютным путём.
		abs, aerr := filepath.Abs(p)
		if aerr != nil {
			return "", false
		}
		return filepath.Clean(abs), true
	}
	return rp, true
}

func resolvedRoots() []string {
	out := make([]string, 0, len(userRoots))
	for _, r := range userRoots {
		if rp, ok := realPath(r); ok {
			out = append(out, rp)
			continue
		}
		out = append(out, filepath.Clean(r))
	}
	return out
}

func under(path string, roots []string) bool {
	path = filepath.Clean(path)
	for _, root := range roots {
		root = filepath.Clean(root)
		if path == root || strings.HasPrefix(path, root+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

// safeDir разрешает только сам ~ и каталоги внутри userRoots.
func safeDir(p string) (string, error) {
	if p == "" {
		p = userRoots[0]
	}
	rp, ok := realPath(p)
	if !ok {
		return "", errNotAllowed
	}
	homeR, _ := realPath(Home)
	if rp == homeR {
		return rp, nil
	}
	if under(rp, resolvedRoots()) {
		if st, err := os.Stat(rp); err == nil && st.IsDir() {
			return rp, nil
		}
	}
	return "", errNotAllowed
}

func regularNonSymlink(p string) bool {
	st, err := os.Lstat(p)
	if err != nil {
		return false
	}
	return st.Mode().IsRegular()
}

// safeFile разрешает только регулярные файлы с нужным расширением внутри userRoots.
func safeFile(p string, exts []string) (string, error) {
	return safeFileExtra(p, exts, "")
}

// safeFileExtra дополнительно разрешает один конкретный путь вне userRoots —
// кадр камеры в /tmp, который QML пишет для распознавания QR.
func safeFileExtra(p string, exts []string, extra string) (string, error) {
	if !okPath(p) {
		return "", errNotAllowed
	}
	if extra != "" && filepath.Clean(p) == filepath.Clean(extra) {
		if !regularNonSymlink(p) {
			return "", errNotAllowed
		}
		return filepath.Clean(p), nil
	}
	rp, ok := realPath(p)
	if !ok || !under(rp, resolvedRoots()) {
		return "", errNotAllowed
	}
	low := strings.ToLower(rp)
	match := false
	for _, e := range exts {
		if strings.HasSuffix(low, e) {
			match = true
			break
		}
	}
	if !match || !regularNonSymlink(rp) {
		return "", errNotAllowed
	}
	return rp, nil
}

// listDir отдаёт подкаталоги и подходящие файлы; в ~ показывает только разрешённые корни.
func listDir(p string) (dirs, files []string, parent string, err error) {
	d, err := safeDir(p)
	if err != nil {
		return nil, nil, "", err
	}
	ents, rerr := os.ReadDir(d)
	if rerr != nil {
		return nil, nil, "", rerr
	}
	homeR, _ := realPath(Home)
	if d == homeR {
		allow := map[string]bool{}
		for _, r := range userRoots {
			allow[filepath.Base(r)] = true
		}
		for _, e := range ents {
			if e.IsDir() && allow[e.Name()] {
				dirs = append(dirs, e.Name())
			}
		}
		sort.Strings(dirs)
		return dirs, nil, d, nil
	}
	for i, e := range ents {
		if i >= 500 {
			break
		}
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		if e.IsDir() {
			dirs = append(dirs, e.Name())
			continue
		}
		low := strings.ToLower(e.Name())
		// Картинки нужны для импорта QR из файла.
		for _, ext := range append(append([]string{}, confExt...), imgExt...) {
			if strings.HasSuffix(low, ext) {
				files = append(files, e.Name())
				break
			}
		}
	}
	sort.Strings(dirs)
	sort.Strings(files)
	parent = filepath.Dir(strings.TrimRight(d, "/"))
	if pr, perr := safeDir(parent); perr == nil {
		parent = pr
	} else {
		parent = homeR
	}
	return dirs, files, parent, nil
}
