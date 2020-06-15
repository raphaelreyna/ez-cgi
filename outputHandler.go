package cgi

import (
	"bufio"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// OutputHandler should handle reading in the client CGI process' output from stdoutRead and
// write out the response to the HTTP client.
// By the time OutputHandler is called, the client CGI process will have already been started and will
// be killed right after OutputHandler returns.
//
// The client CGI process does not need to provide any headers, Handler will provide default Header values.
// If the executable does provide header values, they will overwrite the default values in Header.
// Currently ignored headers: "Location"
type OutputHandler func(w http.ResponseWriter, r *http.Request,
	h *Handler, stdoutReader io.Reader)

// EZOutputHandler sends the entire output of the client process without scanning for headers.
// Always responds with a 200 status code.
var EZOutputHandler OutputHandler = func(w http.ResponseWriter, r *http.Request, h *Handler, stdoutRead io.Reader) {
	for k, vv := range h.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(http.StatusOK)

	linebody := bufio.NewReaderSize(stdoutRead, 1024)
	_, err := io.Copy(w, linebody)
	if err != nil {
		h.logErr("cgi: copy error: %v", err)
		return
	}
}

// EZOutputHandlerReplacer scans the output of the client process for headers which replaces the default header values.
// Stops scanning for headers after encountering the first non-header line.
// The rest of the output is then sent as the response body.
var EZOutputHandlerReplacer OutputHandler = func(w http.ResponseWriter, r *http.Request, h *Handler, stdoutRead io.Reader) {
	internalError := func(err error) {
		w.WriteHeader(http.StatusInternalServerError)
		h.logErr("CGI error: %v", err)
	}

	// readBytes holds the bytes read during header scan but that aren't part of the header.
	// This data will be added to the front of the responses body
	var readBytes []byte
	linebody := bufio.NewReaderSize(stdoutRead, 1024)
	statusCode := 0

	for {
		line, tooBig, err := linebody.ReadLine()
		if tooBig || err == io.EOF {
			break
		}
		if err != nil {
			internalError(err)
			return
		}
		if len(line) == 0 {
			break
		}

		parts := strings.SplitN(string(line), ":", 2)
		if len(parts) < 2 {
			// This line is not a header, add it to the head of the body and break
			readBytes = line
			readBytes = append(line, '\n', '\r')
			break
		}

		k := strings.TrimSpace(parts[0])
		v := strings.TrimSpace(parts[1])

		switch {
		case k == "Status":
			if len(v) < 3 {
				h.logErr("cgi: bogus status (short): %q", v)
				return
			}
			code, err := strconv.Atoi(v[0:3])
			if err != nil {
				h.logErr("cgi: bogus status: %q", v)
				h.logErr("cgi: line was %q", line)
				return
			}
			statusCode = code
		default:
			h.Header.Set(k, v)
		}
	}
	if statusCode == 0 {
		statusCode = http.StatusOK
	}

	for k, vv := range h.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(statusCode)

	// Add back in the beginning portion of the body that was slurped up while scanning for headers.
	if readBytes != nil {
		_, err := w.Write(readBytes)
		if err != nil {
			h.logErr("cgi: copy error: %v", err)
			return
		}
	}

	_, err := io.Copy(w, linebody)
	if err != nil {
		h.logErr("cgi: copy error: %v", err)
		return
	}
}

// DefaultOutputHandler *mostly* mimics the behavior of the net/http/cgi package in the Go standard library.
// The only difference is DefaultOutputHandler does not call on the PathLocationHandler function found in the standard library.
// Currently ignored headers: "Location"
var DefaultOutputHandler OutputHandler = func(w http.ResponseWriter, r *http.Request,
	h *Handler, stdoutRead io.Reader) {
	linebody := bufio.NewReaderSize(stdoutRead, 1024)
	headers := make(http.Header)
	statusCode := 0
	headerLines := 0
	sawBlankLine := false
	for {
		line, isPrefix, err := linebody.ReadLine()
		if isPrefix {
			w.WriteHeader(http.StatusInternalServerError)
			h.logErr("cgi: long header line from subprocess.")
			return
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			h.logErr("cgi: error reading headers: %v", err)
			return
		}
		if len(line) == 0 {
			sawBlankLine = true
			break
		}
		headerLines++
		parts := strings.SplitN(string(line), ":", 2)
		if len(parts) < 2 {
			h.logErr("cgi: bogus header line: %s", string(line))
			continue
		}
		header, val := parts[0], parts[1]
		header = strings.TrimSpace(header)
		val = strings.TrimSpace(val)
		switch {
		case header == "Status":
			if len(val) < 3 {
				h.logErr("cgi: bogus status (short): %q", val)
				return
			}
			code, err := strconv.Atoi(val[0:3])
			if err != nil {
				h.logErr("cgi: bogus status: %q", val)
				h.logErr("cgi: line was %q", line)
				return
			}
			statusCode = code
		default:
			headers.Add(header, val)
		}
	}
	if headerLines == 0 || !sawBlankLine {
		w.WriteHeader(http.StatusInternalServerError)
		h.logErr("cgi: no headers")
		return
	}

	if loc := headers.Get("Location"); loc != "" {
		if statusCode == 0 {
			statusCode = http.StatusFound
		}
	}

	if statusCode == 0 && headers.Get("Content-Type") == "" {
		w.WriteHeader(http.StatusInternalServerError)
		h.logErr("cgi: missing required Content-Type in headers")
		return
	}

	if statusCode == 0 {
		statusCode = http.StatusOK
	}

	for k, vv := range headers {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}

	w.WriteHeader(statusCode)

	_, err := io.Copy(w, linebody)
	if err != nil {
		h.logErr("cgi: copy error: %v", err)
	}
}
