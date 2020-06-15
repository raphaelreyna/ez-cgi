package cmd

import (
	"github.com/raphaelreyna/ez-cgi/pkg/cgi"
	"github.com/spf13/cobra"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

var version string
var versionFlag bool

var (
	noError bool
	port    string

	executable   string
	dir          string
	shell        string
	shellCommand bool

	certFile string
	keyFile  string

	rawHeaders []string
	replace    bool
	conformCGI bool

	envVars []string

	stderr string
)

var RootCmd = &cobra.Command{
	Use:     "ez-cgi [flags]... executable [args]...",
	Version: version,
	Short:   "A friendly and easy to use (almost-)CGI HTTP server.",
	Long: `Start a (almost-)CGI HTTP server.
By default, ez-cgi sends the HTTP client the header 'Content-Type: text/plain'.
If the --replace,-r flag is set and the executable doesn't set any headers, a default header will be set (Content-Type: text/plain).
`,
	Run: run,
}

func SetFlags() {
	RootCmd.Flags().BoolVarP(&versionFlag, "version", "v", false, "Version for ez-cgi.")

	RootCmd.Flags().StringVarP(&port, "port", "p", "8080", "Port to bind to.")

	RootCmd.Flags().BoolVarP(&noError, "quiet", "q", false,
		`Don't show error messages.`,
	)

	RootCmd.Flags().StringVar(&certFile, "tls-cert", "", `Certificate file to use for HTTPS.
Key file must also be provided using the --tls-key flag.`,
	)
	RootCmd.Flags().StringVar(&keyFile, "tls-key", "", `Key file to use for HTTPS.
Cert file must also be provided using the --tls-cert flag.`,
	)

	RootCmd.Flags().StringArrayVarP(&rawHeaders, "header", "H", nil, `HTTP header to send to client.
To allow executable to override header see the --replace flag.
Must be in the form 'KEY: VALUE'.`,
	)
	RootCmd.Flags().BoolVarP(&replace, "replace", "r", false, `Allow executable to replace default header values.
See also: --cgi, -C.`)

	RootCmd.Flags().BoolVarP(&conformCGI, "cgi", "C", false, `Conform to the CGI standard (expect for in the handling of the 'Location' header which is ignored.)
This flag overrides the --replace, -r flag.`,
	)

	RootCmd.Flags().StringVarP(&shell, "shell", "s", "/usr/bin/sh", `Which shell ez-cgi should use when a shell command is passed.
See also: --shell-command, -S.`,
	)
	RootCmd.Flags().BoolVarP(&shellCommand, "shell-command", "S", false, `The argument executable will be interpreted as a shell command.
The command will be passed to the shell set by the --shell, -s flag (sh by default).
This flag is ignored if the --shell, -S flag is not set.
See also: --shell, -S.`,
	)

	RootCmd.Flags().StringArrayVarP(&envVars, "env-var", "e", nil, `Environment variable to pass on to the executable.
Must be in the form 'KEY=VALUE'.`,
	)

	RootCmd.Flags().StringVarP(&stderr, "stderr", "E", "", `Where to redirect executable's stderr.`)

	RootCmd.Flags().StringVarP(&dir, "dir", "d", "", `Working directory for the executable.
Defaults to where ez-cgi was called.`,
	)
}

func run(cmd *cobra.Command, args []string) {
	var err error
	argLen := len(args)
	if argLen == 0 {
		os.Exit(0)
	}
	s := newServer()
	s.port = port

	handler := &cgi.Handler{
		InheritEnv: envVars,
	}

	if shellCommand {
		handler.Path = shell
		handler.Args = []string{"-c", args[0]}
	} else {
		handler.Path = args[0]
		if argLen >= 2 {
			handler.Args = args[1:argLen]
		}
	}

	if stderr != "" {
		handler.Stderr, err = os.Open(stderr)
		defer handler.Stderr.(io.WriteCloser).Close()
		if err != nil {
			log.Printf("error opening stderr: %s", err.Error())
			os.Exit(1)
		}
	}

	header := http.Header{}
	for _, rh := range rawHeaders {
		parts := strings.SplitN(rh, ":", 2)
		if len(parts) < 2 {
			log.Printf("invalid header: %s", rh)
			os.Exit(1)
		}
		k := strings.TrimSpace(parts[0])
		v := strings.TrimSpace(parts[1])
		header.Set(k, v)
	}
	if len(header) != 0 {
		handler.Header = header
	}

	if dir != "" {
		handler.Dir = dir
	} else {
		handler.Dir, err = os.Getwd()
		if err != nil {
			log.Printf("error opening stderr: %s", err.Error())
			os.Exit(1)
		}
	}

	if !noError {
		handler.Logger = log.New(os.Stderr, "\nerror :: ", log.LstdFlags)
	}

	if replace {
		handler.OutputHandler = cgi.EZOutputHandlerReplacer
	}

	if conformCGI {
		handler.OutputHandler = cgi.DefaultOutputHandler
	}

	if handler.OutputHandler == nil {
		handler.OutputHandler = cgi.EZOutputHandler
	}

	server := &http.Server{
		Addr:    ":" + port,
		Handler: handler,
	}

	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigChan
		server.Shutdown(cmd.Context())
		os.Exit(0)
	}()

	if certFile != "" && keyFile != "" {
		server.ListenAndServeTLS(certFile, keyFile)
	} else {
		server.ListenAndServe()
	}
	os.Exit(0)
}

func Execute() {
	SetFlags()
	if err := RootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(0)
	}
}
