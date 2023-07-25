//go:generate go run github.com/abice/go-enum@v0.5.6

package server

import (
	"context"
	"crypto/rsa"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/technopolitica/open-mobility/types"
)

type Env struct {
	db *pgxpool.Pool
}

type authClaims struct {
	jwt.RegisteredClaims
	types.AuthInfo
}

func GetAuthInfo(r *http.Request) (auth types.AuthInfo) {
	ctx := r.Context()
	auth, ok := ctx.Value(ContextKeyAuth).(types.AuthInfo)
	if !ok {
		panic("missing required AuthClaims")
	}
	return
}

// ENUM(auth)
type contextKey int

func parseBearerToken(r *http.Request) (bearerToken string, err error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		err = fmt.Errorf("missing required Authorization header")
		return
	}
	authTokenParts := strings.Split(authHeader, "Bearer ")
	if len(authTokenParts) == 0 {
		err = fmt.Errorf("unsupported or malformed Authorization header (only Bearer scheme is supported)")
		return
	}
	if len(authTokenParts) < 2 {
		err = fmt.Errorf("malformed Authorization header missing bearer token")
		return
	}
	bearerToken = authTokenParts[1]
	return
}

func checkAuthentication(r *http.Request, publicKey *rsa.PublicKey) (authInfo types.AuthInfo, err error) {
	bearerToken, err := parseBearerToken(r)
	if err != nil {
		return
	}
	parser := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Name, jwt.SigningMethodRS384.Name, jwt.SigningMethodRS512.Name}))
	var claims authClaims
	authToken, err := parser.ParseWithClaims(bearerToken, &claims, func(t *jwt.Token) (interface{}, error) {
		return publicKey, nil
	})
	if err != nil {
		err = fmt.Errorf("invalid auth token: %w", err)
		return
	}
	if !authToken.Valid {
		err = fmt.Errorf("invalid auth token")
		return
	}
	authInfo = claims.AuthInfo
	return
}

func authentication(publicKey *rsa.PublicKey) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authInfo, err := checkAuthentication(r, publicKey)
			if err != nil {
				log.Printf("%s", err)
				w.Header().Set("WWW-Authenticate", `Bearer, charset="UTF-8"`)
				w.WriteHeader(401)
				return
			}
			r = r.WithContext(context.WithValue(r.Context(), ContextKeyAuth, authInfo))
			next.ServeHTTP(w, r)
		})
	}
}

// FIXME: probably MUCH better to use JWKS here so we don't have to restart the server to change keys.
func New(db *pgxpool.Pool, publicKey rsa.PublicKey) *chi.Mux {
	router := chi.NewRouter()
	router.Use(middleware.Logger)
	router.Use(middleware.AllowContentType("application/vnd.mds+json"))
	router.Use(middleware.Heartbeat("/health"))
	router.Use(middleware.Timeout(15 * time.Second))
	router.Use(authentication(&publicKey))

	env := Env{db}

	vehiclesRouter := NewVehiclesRouter(&env)
	router.Mount("/vehicles", vehiclesRouter)

	return router
}
