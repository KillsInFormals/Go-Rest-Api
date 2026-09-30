package router

import (
	"net/http"
)

func MainRouter() *http.ServeMux {
	eRouter := execsRouter()
	tRouter := teachersRouter()
	sRouter := studentsRouter()

	sRouter.Handle("/", eRouter)
	tRouter.Handle("/", sRouter)
	return tRouter
}
