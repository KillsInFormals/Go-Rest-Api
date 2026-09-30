package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	mw "restapi/internal/api/middlewares"
	"restapi/internal/api/router"
	"restapi/internal/repository/sqlconnect"
	"restapi/pkg/utils"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	port := os.Getenv("API_PORT")
	cert := "cert.pem"
	key := "key.pem"

	_, err = sqlconnect.ConnectDb()

	if err != nil {
		utils.ErrorHandler(err,"")
		return
	}

	tlsconfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	//rl := mw.NewRateLimiter(5, time.Minute)

	// HPPOptions := mw.HPPOptions{
	// 	CheckQuery:                  true,
	// 	CheckBody:                   true,
	// 	CheckBodyOnlyForContentType: "application/x-www-form-urlencoded",
	// 	Whitelist:                   []string{"sortBy", "sortOrder", "name", "age", "class"},
	// }
	//create custom server

	//secureMux := mw.HPP(HPPOptions)(rl.Middleware(mw.Compression(mw.Cors(mw.ResponseTimeHeader(mw.SecurityHeaders(mw.Cors(mux)))))))
	//secureMux := utils.ApplyMiddlewares(mux, mw.HPP(HPPOptions), mw.Compression, mw.SecurityHeaders, mw.ResponseTimeHeader, rl.Middleware, mw.Cors)
	router := router.MainRouter()
	secureMux := mw.SecurityHeaders(router)

	server := &http.Server{
		Addr: port,
		// Handler:   middlewares.SecurityHeaders(mux),
		Handler:   secureMux,
		TLSConfig: tlsconfig,
	}

	fmt.Println("Server is running on port", port)

	err = server.ListenAndServeTLS(cert, key)
	if err != nil {
		log.Fatalln("Error starting the server:", err)
	}

}
