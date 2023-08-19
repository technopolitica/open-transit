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
	"github.com/technopolitica/open-transit/internal/db"
	"github.com/technopolitica/open-transit/internal/domain"
)

type authClaims struct {
	jwt.RegisteredClaims
	domain.AuthInfo
}

func GetAuthInfo(r *http.Request) (auth domain.AuthInfo) {
	ctx := r.Context()
	auth, ok := ctx.Value(ContextKeyAuth).(domain.AuthInfo)
	if !ok {
		panic("missing required AuthClaims")
	}
	return
}

func GetRepository(r *http.Request) (repo db.Repository) {
	ctx := r.Context()
	repo, ok := ctx.Value(ContextKeyRepository).(db.Repository)
	if !ok {
		panic("missing required repository")
	}
	return
}

// ENUM(auth, repository)
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

func checkAuthentication(r *http.Request, publicKey *rsa.PublicKey) (authInfo domain.AuthInfo, err error) {
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

func database(dbConnPool *pgxpool.Pool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			conn, err := dbConnPool.Acquire(ctx)
			if err != nil {
				log.Printf("failed to acquire database connection: %s", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			defer conn.Release()
			repo, err := db.NewRepository(ctx, conn.Conn())
			if err != nil {
				log.Printf("failed to construct repository: %s", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			r = r.WithContext(context.WithValue(ctx, ContextKeyRepository, repo))
			next.ServeHTTP(w, r)
		})
	}
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

func addHostToRequestURL(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.URL.Host = r.Host
		if r.TLS != nil {
			r.URL.Scheme = "https"
		} else {
			r.URL.Scheme = "http"
		}
		next.ServeHTTP(w, r)
	})
}

// FIXME: probably MUCH better to use JWKS here so we don't have to restart the server to change keys.
func New(db *pgxpool.Pool, publicKey rsa.PublicKey) *chi.Mux {
	router := chi.NewRouter()
	router.Use(middleware.Logger)
	router.Use(middleware.AllowContentType("application/vnd.mds+json"))
	router.Use(middleware.Heartbeat("/health"))
	router.Use(middleware.Timeout(15 * time.Second))
	router.Use(addHostToRequestURL)
	router.Use(authentication(&publicKey))
	router.Use(database(db))

	vehiclesRouter := NewVehiclesRouter()
	router.Mount("/vehicles", vehiclesRouter)

	return router
}
