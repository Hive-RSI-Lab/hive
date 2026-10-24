package gate

import (
	"context"
	"encoding/base64"
	"time"
)

func contextWith(d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), d)
}

// basic is the Authorization value git over HTTPS takes for a GitHub token.
func basic(token string) string {
	return base64.StdEncoding.EncodeToString([]byte("x-access-token:" + token))
}
