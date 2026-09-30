package middlewares

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/golang-jwt/jwt/v5"
)

func JWTMiddleware(next http.Handler) http.Handler {
	fmt.Println("-------------------------JWT MiddleWare---------------------")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("+++++++++++Inside JWT middleware+++++++++++++++++")

		token, err := r.Cookie("Bearer")
		if err != nil {
			http.Error(w, "Authorization Header Missing", http.StatusUnauthorized)
		}
		jwtSecret := os.Getenv("JWT_SECRET")

		parsedToken, err := jwt.Parse(token.Value, func(token *jwt.Token) (interface{}, error) {
			// Don't forget to validate the alg is what you expect:
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
			}

			// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
			return []byte(jwtSecret), nil
		})
		if err != nil {
			if errors.Is(err, jwt.ErrTokenExpired) {
				http.Error(w, "Token Expired", http.StatusUnauthorized)
				return
			} else if errors.Is(err, jwt.ErrTokenMalformed) {
				http.Error(w, "Token Malformed", http.StatusUnauthorized)
				return
			}
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		if parsedToken.Valid {
			log.Println("Valid JWT")
		} else {
			http.Error(w, "Invalid Login token", http.StatusUnauthorized)
			log.Println("Invalid JWT", token.Value)
		}
		fmt.Println("Parsed Token:", parsedToken)
		claims, ok := parsedToken.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, "Invalid Login token", http.StatusUnauthorized)
			log.Println("Invalid JWT", token.Value)
			return
		}

		ctx := context.WithValue(r.Context(), "role", claims["role"])
		ctx = context.WithValue(ctx, "expiredAt", claims["exp"])
		ctx = context.WithValue(ctx, "username", claims["user"])
		ctx = context.WithValue(ctx, "userid", claims["uid"])

		fmt.Println(ctx)

		next.ServeHTTP(w, r.WithContext(ctx))
		fmt.Println("Sent Response from JWT Middleware")

	})
}
