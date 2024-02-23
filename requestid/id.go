package requestid

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type idKey struct{} // key for the context value

func Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uuid := fmt.Sprintf("%X", time.Now().Unix())
		next(w, r.WithContext(WithID(r.Context(), uuid)))
	}
}

func WithID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, idKey{}, id)
}

func GetID(ctx context.Context) string {
	return ctx.Value(idKey{}).(string)
}
