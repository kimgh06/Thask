package cmd

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	cfgpkg "github.com/thask/cli/internal/config"
)

// loginCmd implements the browser-based authorization flow described in
// Phase 12 of the plan: open the configured Thask URL's /cli/auth page in a
// browser, wait for the page to send the freshly-minted API key to a local
// loopback server, then write the token to ~/.thask/config.json. Removes the
// "copy a 64-char string from a web page into a terminal" step.
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate the CLI by opening a browser to your Thask instance",
	Long: `Opens https://<your-thask>/cli/auth in a browser. After you click
Approve there, the server mints a new user_interactive API key and hands it
to a short-lived loopback HTTP server this command spins up locally. The
token is then written to ~/.thask/config.json — no copy/paste required.

Flags:
  --url <url>   Override the Thask URL for this login (also persisted)
  --force       Replace an existing token without prompting`,
	RunE: runLogin,
}

func init() {
	loginCmd.Flags().String("url", "", "Override Thask URL (persisted on success)")
	loginCmd.Flags().Bool("force", false, "Replace an existing token without prompting")
	rootCmd.AddCommand(loginCmd)
}

const (
	loginPortStart   = 7400
	loginPortEnd     = 7500
	loginTimeout     = 5 * time.Minute
	loginStateBytes  = 16
	loginSuccessHTML = `<!DOCTYPE html>
<html><head><title>Thask CLI</title>
<style>body{font:14px system-ui;text-align:center;padding:60px;color:#222}</style></head>
<body><h1>✓ Logged in</h1>
<p>Return to your terminal — you can close this tab.</p></body></html>`
	loginDeniedHTML = `<!DOCTYPE html>
<html><head><title>Thask CLI</title>
<style>body{font:14px system-ui;text-align:center;padding:60px;color:#222}</style></head>
<body><h1>Authorization cancelled</h1>
<p>Return to your terminal.</p></body></html>`
)

type loginResult struct {
	token string
	err   error
}

func runLogin(cmd *cobra.Command, _ []string) error {
	cfg := cfgpkg.Load()

	if u, _ := cmd.Flags().GetString("url"); u != "" {
		cfg.URL = strings.TrimRight(cfgpkg.NormalizeURL(u), "/")
	}
	if cfg.URL == "" {
		return fmt.Errorf("URL is not configured — run `thask config set url <url>` or pass `--login --url <url>`")
	}

	force, _ := cmd.Flags().GetBool("force")
	if cfg.Token != "" && !force {
		fmt.Fprintf(os.Stderr, "Already logged in (token %s). Re-run with --force to replace.\n", cfg.MaskedToken())
		return nil
	}

	state, err := randomStateHex()
	if err != nil {
		return fmt.Errorf("generate state: %w", err)
	}

	listener, port, err := bindLoopback(loginPortStart, loginPortEnd)
	if err != nil {
		return fmt.Errorf("no free local port in %d-%d", loginPortStart, loginPortEnd)
	}

	resultCh := make(chan loginResult, 1)
	srv := &http.Server{
		Handler:      buildLoginHandler(state, resultCh),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
	go srv.Serve(listener) //nolint:errcheck // closed via Shutdown

	authURL := buildAuthURL(cfg.URL, port, state)
	fmt.Fprintln(os.Stderr, "Opening browser… if it does not open, paste this URL:")
	fmt.Fprintln(os.Stderr, "  ", authURL)
	if err := browser.OpenURL(authURL); err != nil {
		// Non-fatal — the URL is already on stderr for manual fallback.
		fmt.Fprintln(os.Stderr, "(could not auto-open browser:", err, "— paste the URL above)")
	}

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()

	select {
	case res := <-resultCh:
		if res.err != nil {
			return res.err
		}
		cfg.Token = res.token
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("save config: %w", err)
		}
		fmt.Fprintf(os.Stderr, "✓ Logged in. Token saved (%s).\n", maskToken(res.token))
		return nil
	case <-time.After(loginTimeout):
		return fmt.Errorf("no response within %s — re-run `thask login`", loginTimeout)
	}
}

// buildLoginHandler returns the handler the browser hits via the redirect
// from /cli/auth. Validates state, captures token or error reason, sends a
// terminal-friendly HTML page back, then signals the main goroutine.
func buildLoginHandler(expectedState string, resultCh chan<- loginResult) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		gotState := q.Get("state")
		if gotState != expectedState {
			http.Error(w, "state mismatch", http.StatusBadRequest)
			select {
			case resultCh <- loginResult{err: fmt.Errorf("security check failed (state mismatch) — possible CSRF, aborting")}:
			default:
			}
			return
		}
		if reason := q.Get("error"); reason != "" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprint(w, loginDeniedHTML)
			select {
			case resultCh <- loginResult{err: fmt.Errorf("authorization denied (%s)", reason)}:
			default:
			}
			return
		}
		token := q.Get("token")
		if token == "" {
			http.Error(w, "missing token", http.StatusBadRequest)
			select {
			case resultCh <- loginResult{err: fmt.Errorf("callback missing ?token parameter")}:
			default:
			}
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, loginSuccessHTML)
		select {
		case resultCh <- loginResult{token: token}:
		default:
		}
	})
}

// bindLoopback tries ports in [start,end] on 127.0.0.1 and returns the first
// successful listener + its port. IPv4 loopback only — never exposes the
// callback to the LAN.
func bindLoopback(start, end int) (net.Listener, int, error) {
	for p := start; p <= end; p++ {
		l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", p))
		if err == nil {
			return l, p, nil
		}
	}
	return nil, 0, fmt.Errorf("no free port")
}

func randomStateHex() (string, error) {
	var b [loginStateBytes]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func buildAuthURL(base string, port int, state string) string {
	q := url.Values{}
	q.Set("callback_port", fmt.Sprintf("%d", port))
	q.Set("state", state)
	return strings.TrimRight(base, "/") + "/cli/auth?" + q.Encode()
}

func maskToken(t string) string {
	if len(t) <= 12 {
		return t
	}
	return t[:12] + "…"
}
