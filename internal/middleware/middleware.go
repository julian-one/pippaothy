package middleware

import (
	"context"
	"net/http"
	"pippaothy/internal/auth"
	"pippaothy/internal/users"

	"github.com/jmoiron/sqlx"
)

type contextKey string

const userContextKey contextKey = "authenicatedUser"

func IsAuthenticated(db *sqlx.DB, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var user *users.User

		cookie, err := r.Cookie("session_token")
		if err == nil {
			user, err = auth.GetSession(db, cookie.Value)
			if err != nil {
				user = nil
			}
		}

		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func GetCtxUser(r *http.Request) *users.User {
	user, _ := r.Context().Value(userContextKey).(*users.User)
	return user
}
