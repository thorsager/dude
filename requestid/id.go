package requestid

import (
	"context"
	"encoding/base64"
	"github.com/google/uuid"
	"net/http"
)

type idKey struct{} // key for the context value

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uid := uuid.Must(uuid.NewUUID()) // generate a new UUID version 1 is fine for this application
		rqid := base64.RawStdEncoding.EncodeToString(uid[:])
		next.ServeHTTP(w, r.WithContext(withID(r.Context(), rqid)))
	})
}

func withID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, idKey{}, id)
}

func GetID(ctx context.Context) string {
	return ctx.Value(idKey{}).(string)
}
