## ez-cgi

A friendly and easy to use (almost-)CGI HTTP server.


### Installation

There are multiple ways of obtaining ez-cgi:

##### Brew
```bash
brew tap raphaelreyna/homebrew-repo
brew install ez-cgi
```

##### Go get
```bash
go get -u -v github.com/raphaelreyna/ez-cgi/cmd
```

##### Compiling from source
```bash
git clone github.com/raphaelreyna/ez-cgi
cd ez-cgi/cmd
sudo make install
```


### Synopsis


Start a (almost-)CGI HTTP server.
By default, ez-cgi sends the HTTP client the header 'Content-Type: text/plain'.
If the --replace,-r flag is set and the executable doesn't set any headers, a default header will be set (Content-Type: text/plain).


```
ez-cgi [flags]... executable [args]...
```

### Options

```
  -C, --cgi                   
                              Conform to the CGI standard (expect for in the handling of the 'Location' header which is ignored.)
                              This flag overrides the --replace, -r flag.
  -d, --dir string            
                              Working directory for the executable.
                              Defaults to where ez-cgi was called.
  -e, --env-var stringArray   
                              Environment variable to pass on to the executable.
                              Must be in the form 'KEY=VALUE'.
  -H, --header stringArray    
                              HTTP header to send to client.
                              To allow executable to override header see the --replace flag.
                              Must be in the form 'KEY: VALUE'.
  -h, --help                  help for ez-cgi
  -p, --port string           
                              Port to bind to. (default "8080")
  -q, --quiet                 
                              Don't show error messages.
  -r, --replace               
                              Allow executable to replace default header values.
                              See also: --cgi, -C.
  -s, --shell string          
                              Which shell ez-cgi should use when a shell command is passed.
                              See also: --shell-command, -S. (default "/usr/bin/sh")
  -S, --shell-command         
                              The argument executable will be interpreted as a shell command.
                              The command will be passed to the shell set by the --shell, -s flag (sh by default).
                              This flag is ignored if the --shell, -S flag is not set.
                              See also: --shell, -S.
  -E, --stderr string         
                              Where to redirect executable's stderr.
      --tls-cert string       
                              Certificate file to use for HTTPS.
                              Key file must also be provided using the --tls-key flag.
      --tls-key string        
                              Key file to use for HTTPS.
                              Cert file must also be provided using the --tls-cert flag.
  -v, --version               
                              Version for ez-cgi.
```

###### Auto generated by spf13/cobra on 15-Jun-2020
