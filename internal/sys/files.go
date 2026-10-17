package sys

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// HasMarkerLine reports whether data has a line equal to marker, like
// `grep -Fqx`, ignoring a trailing CR. It also matches marker lines
// embedded in a binary.
func HasMarkerLine(data []byte, marker string) bool {
	ok, _ := hasMarkerLine(bytes.NewReader(data), marker)
	return ok
}

// hasMarkerLine scans r line by line without holding it in memory, so a
// multi-megabyte binary costs one buffer.
func hasMarkerLine(r io.Reader, marker string) (bool, error) {
	br := bufio.NewReaderSize(r, 64<<10)
	long := false
	for {
		line, err := br.ReadSlice('\n')
		if errors.Is(err, bufio.ErrBufferFull) {
			// Too long to be the marker: skip to the next newline.
			long = true
			continue
		}
		if !long && string(bytes.TrimSuffix(bytes.TrimSuffix(line, []byte("\n")), []byte("\r"))) == marker {
			return true, nil
		}
		long = false
		if err == io.EOF {
			return false, nil
		}
		if err != nil {
			return false, err
		}
	}
}

// FileHasMarkerLine is HasMarkerLine for a host file, streamed.
func FileHasMarkerLine(path, marker string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	ok, err := hasMarkerLine(f, marker)
	return err == nil && ok
}

// Owned reports whether path is a regular file (not a symlink) carrying the
// marker line: only such files may be replaced or removed.
func (e *Env) Owned(path, marker string) bool {
	f, err := os.OpenFile(e.P(path), os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return false
	}
	defer f.Close()
	if fi, err := f.Stat(); err != nil || !fi.Mode().IsRegular() {
		return false
	}
	ok, err := hasMarkerLine(f, marker)
	return err == nil && ok
}

// Exists reports whether path exists, a dangling symlink included.
func (e *Env) Exists(path string) bool {
	_, err := os.Lstat(e.P(path))
	return err == nil
}

// IsExecutable reports whether path is a file with an execute bit.
func (e *Env) IsExecutable(path string) bool {
	fi, err := os.Stat(e.P(path))
	return err == nil && fi.Mode().IsRegular() && fi.Mode().Perm()&0o111 != 0
}

// IsFile reports whether path is a regular file (symlinks followed).
func (e *Env) IsFile(path string) bool {
	fi, err := os.Stat(e.P(path))
	return err == nil && fi.Mode().IsRegular()
}

// ReadFile reads a router file.
func (e *Env) ReadFile(path string) ([]byte, error) { return os.ReadFile(e.P(path)) }

// Remove deletes a file; a missing file is not an error.
func (e *Env) Remove(path string) error {
	err := os.Remove(e.P(path))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

// WriteAtomic writes data to a temporary file next to path and renames it,
// so a running reader keeps the old inode.
func (e *Env) WriteAtomic(path string, data []byte, perm fs.FileMode) error {
	return e.WriteAtomicFrom(path, bytes.NewReader(data), perm)
}

// WriteAtomicFrom is WriteAtomic with the content streamed from r.
func (e *Env) WriteAtomicFrom(path string, r io.Reader, perm fs.FileMode) error {
	real := e.P(path)
	if err := os.MkdirAll(filepath.Dir(real), 0o755); err != nil {
		return err
	}
	// The name is predictable: drop whatever sits there (a symlink is
	// removed, not followed) and create it exclusively.
	tmp := fmt.Sprintf("%s.tmp.%d", real, os.Getpid())
	if err := os.Remove(tmp); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, perm)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, r)
	// OpenFile honours umask; set the mode explicitly.
	if err == nil {
		err = f.Chmod(perm)
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(tmp, real)
	}
	if err != nil {
		os.Remove(tmp)
	}
	return err
}

// OpenOwn opens path for writing without following symlinks. Anything at
// path that is not a regular file of ours with a single link (a symlink or
// hard link planted in a shared /tmp) is removed and created afresh, so
// root never writes through it; a directory is an error. O_TRUNC is
// applied only after that check.
func (e *Env) OpenOwn(path string, flag int, perm fs.FileMode) (*os.File, error) {
	real := e.P(path)
	if err := os.MkdirAll(filepath.Dir(real), 0o755); err != nil {
		return nil, err
	}
	trunc := flag&os.O_TRUNC != 0
	flag = flag&^os.O_TRUNC | syscall.O_NOFOLLOW | syscall.O_NONBLOCK
	f, err := os.OpenFile(real, flag|os.O_CREATE, perm)
	switch {
	case err == nil && ownFile(f):
	case err == nil || errors.Is(err, syscall.ELOOP):
		if f != nil {
			f.Close()
		}
		if err := os.Remove(real); err != nil {
			return nil, err
		}
		if f, err = os.OpenFile(real, flag|os.O_CREATE|os.O_EXCL, perm); err != nil {
			return nil, err
		}
	default:
		return nil, err
	}
	if trunc {
		if err := f.Truncate(0); err != nil {
			f.Close()
			return nil, err
		}
	}
	return f, nil
}

// ownFile reports whether f is a regular file owned by this process's user
// with no other names.
func ownFile(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil || !fi.Mode().IsRegular() {
		return false
	}
	st, ok := fi.Sys().(*syscall.Stat_t)
	return ok && st.Nlink == 1 && int(st.Uid) == os.Geteuid()
}

// WriteInPlace truncates and rewrites an existing file, keeping its inode,
// owner and mode (like `cat new > file`).
func (e *Env) WriteInPlace(path string, data []byte) error {
	f, err := os.OpenFile(e.P(path), os.O_WRONLY|os.O_TRUNC, 0)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

const tempChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

// CreateTemp creates a new file named prefix + 6 random characters, like
// `mktemp prefixXXXXXX`, and returns its router path.
func (e *Env) CreateTemp(prefix string, data []byte, perm fs.FileMode) (string, error) {
	for range 100 {
		b := make([]byte, 6)
		if _, err := rand.Read(b); err != nil {
			return "", err
		}
		for i := range b {
			b[i] = tempChars[int(b[i])%len(tempChars)]
		}
		path := prefix + string(b)
		f, err := os.OpenFile(e.P(path), os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		_, werr := f.Write(data)
		cerr := f.Close()
		if werr != nil || cerr != nil {
			os.Remove(e.P(path))
			return "", errors.Join(werr, cerr)
		}
		return path, nil
	}
	return "", fmt.Errorf("не удалось создать временный файл %sXXXXXX", prefix)
}

// Backup copies path to prefix+XXXXXX keeping its mode and mtime (`cp -p`).
func (e *Env) Backup(path, prefix string) (string, error) {
	fi, err := os.Stat(e.P(path))
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(e.P(path))
	if err != nil {
		return "", err
	}
	b, err := e.CreateTemp(prefix, data, fi.Mode().Perm())
	if err != nil {
		return "", err
	}
	if err := os.Chmod(e.P(b), fi.Mode().Perm()); err != nil {
		return b, err
	}
	return b, os.Chtimes(e.P(b), fi.ModTime(), fi.ModTime())
}

// ReadPID reads a PID file; 0 when missing or malformed.
func (e *Env) ReadPID(path string) int {
	data, err := os.ReadFile(e.P(path))
	if err != nil {
		return 0
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0
	}
	return pid
}

// FSType returns the filesystem type holding path, checking its deepest
// existing ancestor against /proc/mounts.
func (e *Env) FSType(path string) string {
	p := filepath.Clean(path)
	for {
		if fi, err := os.Stat(e.P(p)); err == nil && fi.IsDir() {
			break
		}
		if p == "/" {
			break
		}
		p = filepath.Dir(p)
	}
	f, err := os.Open(e.P("/proc/mounts"))
	if err != nil {
		return ""
	}
	defer f.Close()
	best, typ := -1, ""
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fl := strings.Fields(sc.Text())
		if len(fl) < 3 {
			continue
		}
		m := strings.ReplaceAll(fl[1], `\040`, " ")
		if (m == "/" || p == m || strings.HasPrefix(p, m+"/")) && len(m) >= best {
			best, typ = len(m), fl[2]
		}
	}
	return typ
}
