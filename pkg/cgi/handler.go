// Package cgi flexibly implements the CGI as specified in RFC 3875.
// Allows for non-CGI conforming executables to be used and provides default headers.
// The executable set to handle the HTTP requests is always provided with the environment variables described in
// RFC 3875 Sections 4.1.2 - 4.1.5, 4.1.7 - 4.1.9, and 4.1.12 - 4.1.17.
// The handling of the executables standard output is handled by a user provided function.
// A lot of this code is copied straight from the Go standard library: https://golang.org/src/net/http/cgi/host.go
package cgi

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var portRegex = regexp.MustCompile(`:([0-9]+)$`)

var osDefaultInheritEnv = map[string][]string{
	"darwin":  {"DYLD_LIBRARY_PATH"},
	"freebsd": {"LD_LIBRARY_PATH"},
	"hpux":    {"LD_LIBRARY_PATH", "SHLIB_PATH"},
	"irix":    {"LD_LIBRARY_PATH", "LD_LIBRARYN32_PATH", "LD_LIBRARY64_PATH"},
	"linux":   {"LD_LIBRARY_PATH"},
	"openbsd": {"LD_LIBRARY_PATH"},
	"solaris": {"LD_LIBRARY_PATH", "LD_LIBRARY_PATH_32", "LD_LIBRARY_PATH_64"},
	"windows": {"SystemRoot", "COMSPEC", "PATHEXT", "WINDIR"},
}

// Handler runs an executable in a subprocess with an almost CGI environment.
// Currently ignored headers: "Location"
type Handler struct {
	Path string

	Root string

	Name string // value to use for SERVER_SOFTWARE env var
	Port string

	Dir string

	InheritEnv []string
	Logger     *log.Logger
	Args       []string
	Stderr     io.Writer

	// Header contains header values that should be used by default.
	// If the client CGI process writes a header to its stdout thats already in Header, it will be replaced.
	Header http.Header
	// OutputHandler takes care of responding the HTTP client based on the CGI client processes output.
	OutputHandler OutputHandler
}

func (h *Handler) logErr(format string, v ...interface{}) {
	if h.Logger != nil {
		h.Logger.Printf(format, v...)
	} else {
		log.Printf(format, v...)
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.Root == "" {
		h.Root = "/"
	}
	if h.Name == "" {
		h.Name = "go"
	}
	if h.Stderr == nil {
		h.Stderr = os.Stderr
	}
	if h.Header == nil {
		h.Header = http.Header{
			"Content-Type": []string{"text/plain"},
		}
	}
	if h.OutputHandler == nil {
		h.OutputHandler = DefaultOutputHandler
	}

	if len(r.TransferEncoding) > 0 && r.TransferEncoding[0] == "chunked" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Chunked request bodies are not supported by CGI."))
		return
	}

	pathInfo := r.URL.Path
	if h.Root != "/" && strings.HasPrefix(pathInfo, h.Root) {
		pathInfo = pathInfo[len(h.Root):]
	}

	port := "8080"

	if matches := portRegex.FindStringSubmatch(r.Host); len(matches) != 0 {

		port = matches[1]

	}

	env := []string{
		"SERVER_SOFTWARE=" + h.Name,
		"SERVER_NAME=" + r.Host,
		"SERVER_PROTOCOL=HTTP/1.1",
		"HTTP_HOST=" + r.Host,
		"GATEWAY_INTERFACE=CGI/1.1",
		"REQUEST_METHOD=" + r.Method,
		"QUERY_STRING=" + r.URL.RawQuery,
		"REQUEST_URI=" + r.URL.RequestURI(),
		"PATH_INFO=" + pathInfo,
		"SCRIPT_NAME=" + h.Root,
		"SCRIPT_FILENAME=" + h.Path,
		"SERVER_PORT=" + port,
	}

	if remoteIP, remotePort, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		env = append(env, "REMOTE_ADDR="+remoteIP, "REMOTE_HOST="+remoteIP, "REMOTE_PORT="+remotePort)
	} else {
		env = append(env, "REMOTE_ADDR="+r.RemoteAddr, "REMOTE_HOST="+r.RemoteAddr)
	}

	if r.TLS != nil {
		env = append(env, "HTTPS=on")
	}

	for k, v := range r.Header {
		k = strings.Map(upperCaseAndUnderscore, k)
		if k == "PROXY" {
			continue
		}
		joinStr := ", "
		if k == "COOKIE" {
			joinStr = "; "
		}
		env = append(env, "HTTP_"+k+"="+strings.Join(v, joinStr))
	}

	if r.ContentLength > 0 {
		env = append(env, fmt.Sprintf("CONTENT_LENGTH=%d", r.ContentLength))
	}
	if ctype := r.Header.Get("Content-Type"); ctype != "" {
		env = append(env, "CONTENT_TYPE="+ctype)
	}

	envPath := os.Getenv("PATH")
	if envPath == "" {
		envPath = "/bin:/usr/bin:/usr/ucb:/usr/bsd:/usr/local/bin"
	}
	env = append(env, "PATH="+envPath)

	for _, e := range h.InheritEnv {
		if v := os.Getenv(e); v != "" {
			env = append(env, e+"="+v)
		}
	}

	env = removeLeadingDuplicates(env)

	var cwd, path string
	if h.Dir != "" {
		path = h.Path
		cwd = h.Dir
	} else {
		cwd, path = filepath.Split(h.Path)
	}
	if cwd == "" {
		cwd = "."
	}

	internalError := func(err error) {
		w.WriteHeader(http.StatusInternalServerError)
		h.logErr("CGI error: %v", err)
	}

	cmd := &exec.Cmd{
		Path:   path,
		Args:   append([]string{h.Path}, h.Args...),
		Dir:    cwd,
		Env:    env,
		Stderr: h.Stderr,
	}

	if r.ContentLength != 0 {
		cmd.Stdin = r.Body
	}
	stdoutRead, err := cmd.StdoutPipe()
	if err != nil {
		internalError(err)
		return
	}
	err = cmd.Start()
	if err != nil {
		internalError(err)
		return
	}

	defer cmd.Wait()
	defer stdoutRead.Close()

	h.OutputHandler(w, r, h, stdoutRead)

	// Make sure the process is good and dead before exiting
	cmd.Process.Kill()
}

func removeLeadingDuplicates(env []string) (ret []string) {
	for i, e := range env {
		found := false
		if eq := strings.IndexByte(e, '='); eq != -1 {
			keq := e[:eq+1]
			for _, e2 := range env[i+1:] {
				if strings.HasPrefix(e2, keq) {
					found = true
					break
				}
			}
		}
		if !found {
			ret = append(ret, e)
		}
	}
	return
}

func upperCaseAndUnderscore(r rune) rune {
	switch {
	case r >= 'a' && r <= 'z':
		return r - ('a' - 'A')
	case r == '-':
		return '_'
	case r == '=':
		return '_'
	}
	return r
}
