package oauth2

import (
	"context"
	"fmt"
	"net"
	"net/http"
)

// StartCallbackServer starts a temporary HTTP server on a random port to receive
// the OAuth2 callback with the authorization code. It blocks until a code is
// received or the context is cancelled.
func StartCallbackServer(ctx context.Context) (code string, port int, err error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", 0, fmt.Errorf("starting callback listener: %w", err)
	}
	port = listener.Addr().(*net.TCPAddr).Port

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		c := r.URL.Query().Get("code")
		if c == "" {
			errMsg := r.URL.Query().Get("error")
			if errMsg == "" {
				errMsg = "no code in callback"
			}
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "<html><body><h1>Error</h1><p>%s</p></body></html>", errMsg)
			errCh <- fmt.Errorf("OAuth2 callback error: %s", errMsg)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "<html><body><h1>Authorization successful!</h1><p>You can close this tab and return to gottp.</p></body></html>")
		codeCh <- c
	})

	server := &http.Server{Handler: mux}

	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case code = <-codeCh:
		_ = server.Shutdown(context.Background())
		return code, port, nil
	case err = <-errCh:
		_ = server.Shutdown(context.Background())
		return "", port, err
	case <-ctx.Done():
		_ = server.Shutdown(context.Background())
		return "", port, ctx.Err()
	}
}
