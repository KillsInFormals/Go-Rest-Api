package middlewares

import (
	"fmt"
	"net/http"
	"time"
)

func ResponseTimeHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		fmt.Println("Request recieved in middleware")

		wrappedWriter := &responseWriter{ResponseWriter: w, status: http.StatusOK}

		
		next.ServeHTTP(wrappedWriter, r)
		duration:= time.Since(start)

		wrappedWriter.Header().Set("X-Response-Time",duration.String())
		

		

		fmt.Printf("Method:%v  URL:%v  Duration:%v  Status:%v\n", r.Method, r.URL, duration.String(), wrappedWriter.status)
		fmt.Println("Request sent from middleware")
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}
