package main

import (
	"github.com/gorilla/mux"
	"net/http"
	_"net/http"

	_"github.com/gorilla/mux"
)

func main() {
	r := mux.NewRouter()

	// Страница по умолчанию для нашего сайта это простой html.
	r.Handle("/", http.FileServer(http.Dir("./views/")))

	// Статику (картинки, скрипти, стили) будем раздавать
	// по определенному роуту /static/{file}
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/",
		http.FileServer(http.Dir("./static/"))))

	r.Handle("/get-tokens", NotImplemented).Methods("GET")
	r.Handle("/refresh", NotImplemented).Methods("POST")
	r.Handle("/delete-token", NotImplemented).Methods("DELETE")
	r.Handle("/delete-tokens", NotImplemented).Methods("DELETE")

	http.ListenAndServe(":3000", r)
}

var NotImplemented = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
	w.Write([]byte("Not Implemented"))
})
