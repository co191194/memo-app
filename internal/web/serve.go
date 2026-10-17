package web

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"
)

func Serve(ctx context.Context, ln net.Listener, handler http.Handler, shutdownTimeout time.Duration) error {

	srv := &http.Server{Handler: handler}

	c := make(chan error, 1)
	go func() {
		c <- srv.Serve(ln)
	}()

	select {
	case err := <-c:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		} else {
			return err
		}
	case <-ctx.Done():
		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)

		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			srv.Close()
		}

		return nil
	}
}
