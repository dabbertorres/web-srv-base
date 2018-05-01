package main

import (
	"io/ioutil"
	"net/http"

)

func staticFileHandler(filepath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		buf, err := ioutil.ReadFile(filepath)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.Write(buf)
		}
	}
}
