package graphql

import (
	"context"
	"net/http"
	"strings"
	"time"

	"client-services/internal/domain"
	"client-services/internal/transport/graphql/convert"
	"github.com/99designs/gqlgen/graphql/handler/transport"
)

func websocketInit(authenticate func(context.Context, string) (context.Context, error)) transport.WebsocketInitFunc {
	return func(ctx context.Context, payload transport.InitPayload) (context.Context, *transport.InitPayload, error) {
		found := false
		for key, value := range payload {
			if !strings.EqualFold(key, "Authorization") {
				continue
			}
			header, ok := value.(string)
			if found || !ok || authenticate == nil {
				return ctx, nil, convert.Error(domain.ErrUnauthenticated)
			}
			found = true
			authenticated, err := authenticate(ctx, header)
			if err != nil {
				return ctx, nil, convert.Error(err)
			}
			ctx = authenticated
		}
		return ctx, nil, nil
	}
}

type websocketImplementation struct {
	transport.CoderWebsocketImplementation
}

func (w websocketImplementation) Accept(rw http.ResponseWriter, r *http.Request, o transport.WebsocketAcceptOptions) (transport.WebsocketConn, error) {
	conn, err := w.CoderWebsocketImplementation.Accept(rw, r, o)
	if err != nil {
		return nil, err
	}
	return &boundedWebsocket{WebsocketConn: conn}, nil
}

type boundedWebsocket struct{ transport.WebsocketConn }

func (c *boundedWebsocket) WriteJSON(value any) error {
	timer := time.AfterFunc(5*time.Second, func() { _ = c.WebsocketConn.Close() })
	defer timer.Stop()
	return c.WebsocketConn.WriteJSON(value)
}
func (c *boundedWebsocket) SetReadLimit(n int64) {
	if conn, ok := c.WebsocketConn.(transport.WebsocketReadLimiter); ok {
		conn.SetReadLimit(n)
	}
}
func (c *boundedWebsocket) SetReadDeadline(t time.Time) error {
	if conn, ok := c.WebsocketConn.(transport.WebsocketReadDeadliner); ok {
		return conn.SetReadDeadline(t)
	}
	return nil
}
