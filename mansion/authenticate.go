package mansion

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/go-errors/errors"
	"github.com/modulesio/butler/art"
	"github.com/modulesio/butler/comm"
	"github.com/itchio/go-itchio"
)

// read+write for owner, no permissions for others
const keyFileMode = 0600

const (
	authHTML = `
        <!DOCTYPE html>
        <html>
        <head>
          <link href="https://fonts.googleapis.com/css?family=Lato:400,700" rel="stylesheet" type="text/css">
          <style>
            body {
              text-align: center;
              margin: 50px 0;
            }

            p {
              line-height: 1.6;
              font-size: 18px;
              font-family: Lato, sans-serif;
            }

            a, a:active, a:visited, a:hover {
              color: #FA5B5B;
            }

            /* A a pastel rainbow palette */
            @keyframes rainbow {
              from { color: #FFB3BA; }
              25%  { color: #FFDFBA; }
              50%  { color: #FFFFBA; }
              75%  { color: #BAFFC9; }
              to   { color: #BAE1FF; }
            }

            pre {
              animation: rainbow alternate 5s infinite linear;
              background: #1C1C1D;
              padding: 2em 0;
              line-height: 1.3;
              font-size: 16px;
              color: #FFB3BA;
              text-shadow: 0 0 20px;
              color: white;
            }
          </style>
        </head>
        <body>
          <pre id="art"></pre>
          <p id="message">
            Authenticating...
          </p>
          <script>
          'use strict'
          var key = location.hash.replace(/^#/, '')
          location.hash = 'ok'
          var xhr = new XMLHttpRequest()
          var $message = document.querySelector("#message")
          var $art = document.querySelector("#art")
          xhr.onload = function () {
            $art.innerHTML = xhr.responseText
            $message.innerHTML = "You're successfully authenticated! You can close this page."
          }
          xhr.onerror = function () {
            $message.innerHTML = "Copy the following code back in your terminal: " + key
          }
          xhr.open("POST", "/oauth/callback/" + key)
          xhr.send()
          </script>
        </body>
      </html>`
)

var callbackRe = regexp.MustCompile(`^\/oauth\/callback\/(.*)$`)

const environmentApiKey = "BUTLER_API_KEY"

func (ctx *Context) HasSavedCredentials() bool {
	// environment has priority
	if os.Getenv(environmentApiKey) != "" {
		return true
	}

	// then file at usual or specified path
	var identity = ctx.Identity
	_, err := os.Lstat(identity)

	exists := !os.IsNotExist(err)
	return exists
}

func readKeyFile(path string) (string, error) {
	stats, err := os.Lstat(path)

	if err != nil && os.IsNotExist(err) {
		// no key file
		return "", nil
	}

	if stats.Mode()&077 > 0 {
		if runtime.GOOS == "windows" {
			// windows won't let you 0600, because it's ACL-based
			// we can make it 0644, and go will report 0666, but
			// it doesn't matter since other users can't access it anyway.
			// empirical evidence: https://github.com/itchio/butler/issues/65
		} else {
			comm.Logf("[Warning] Key file had wrong permissions (%#o), resetting to %#o\n", stats.Mode()&0777, keyFileMode)
			err = os.Chmod(path, keyFileMode)
			if err != nil {
				comm.Logf("[Warning] Couldn't chmod keyfile: %s\n", err.Error())
			}
		}
	}

	buf, err := ioutil.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return "", errors.Wrap(err, 1)
	}

	return strings.TrimSpace(string(buf)), nil
}

func writeKeyFile(path string, key string) error {
	return ioutil.WriteFile(path, []byte(key), os.FileMode(keyFileMode))
}

func (ctx *Context) AuthenticateViaOauth() (*itchio.Client, error) {
	var err error
	var identity = ctx.Identity
	var key string

	makeClient := func(key string) *itchio.Client {
		client := itchio.ClientWithKey(key)
		client.SetServer(ctx.Address)
		client.UserAgent = ctx.UserAgent()
		return client
	}

	envKey := os.Getenv(environmentApiKey)
	if envKey != "" {
		return makeClient(envKey), nil
	}

	key, err = readKeyFile(identity)
	if err != nil {
		return nil, errors.Wrap(err, 1)
	}

	if key == "" {
		done := make(chan string)
		errs := make(chan error)

		handler := func(w http.ResponseWriter, r *http.Request) {
			matches := callbackRe.FindStringSubmatch(r.RequestURI)
			if matches != nil {
				client := makeClient(matches[1])
				client.WharfStatus()

				w.Header().Set("Content-Type", "text/plain")
				fmt.Fprintf(w, art.ItchLogo)
				done <- matches[1]
				return
			}

			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, "%s", authHTML)
		}

		http.HandleFunc("/", handler)

		// if we're running `butler login` remotely, we're asking the user to copy-paste
		var addr = "127.0.0.1:226"
		var doManualOauth = os.Getenv("BUTLER_MANUAL_OAUTH") == "1"

		if !doManualOauth {
			var listener net.Listener
			listener, err = net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				return nil, errors.Wrap(err, 1)
			}

			addr = listener.Addr().String()

			go func() {
				err = http.Serve(listener, nil)
				if err != nil {
					errs <- errors.Wrap(err, 1)
				}
			}()
		}

		form := url.Values{}
		form.Add("client_id", "butler")
		form.Add("scope", "wharf")
		form.Add("response_type", "token")
		form.Add("redirect_uri", fmt.Sprintf("http://%s/oauth/callback", addr))
		query := form.Encode()

		uri := fmt.Sprintf("%s/user/oauth?%s", ctx.Address, query)

		comm.Login(uri)

		go func() {
			s := bufio.NewScanner(os.Stdin)
			for s.Scan() {
				line := strings.TrimSpace(s.Text())
				u, err := url.Parse(line)
				if err != nil {
					// not a valid url
					continue
				}

				if u.Fragment != "" {
					// user pasted the url!
					done <- u.Fragment
					return
				}
			}
		}()

		select {
		case err = <-errs:
			return nil, errors.Wrap(err, 1)
		case key = <-done:
			err = nil

			client := makeClient(key)
			_, err = client.WharfStatus()
			if err != nil {
				return nil, errors.Wrap(err, 1)
			}

			comm.Logf("\nAuthenticated successfully! Saving key in %s...\n", identity)

			err = os.MkdirAll(filepath.Dir(identity), os.FileMode(0755))
			if err != nil {
				comm.Logf("\nCould not create directory for storing API key: %s\n\n", errors.Wrap(err, 1).Error())
				err = nil
			} else {
				err = writeKeyFile(identity, key)
				if err != nil {
					comm.Logf("\nCould not save API key: %s\n\n", errors.Wrap(err, 1).Error())
					err = nil
				}
			}
		}
	}

	if err != nil {
		err = errors.Wrap(err, 1)
	}
	return makeClient(key), nil
}
