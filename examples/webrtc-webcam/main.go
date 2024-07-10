package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
)

type OfferJson struct {
	Offer string `json:"offer"`
}

type ErrorJson struct {
	Error string `json:"error"`
}

func sendError(w http.ResponseWriter, err error) {
	errj := &ErrorJson{
		Error: err.Error(),
	}
	b, _ := json.Marshal(errj)
	w.WriteHeader(http.StatusBadRequest)
	w.Write(b)
}

func main() {
	port := flag.String("p", "8100", "port to serve on")
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exPath := filepath.Dir(ex)
	static := flag.String("s", path.Join(exPath, "static"), "the directory for JS and CSS files")
	flag.Parse()

	mux := http.NewServeMux()
	wr := NewWebrtc()
	mux.Handle("/", http.FileServer(http.Dir(*static)))
	mux.HandleFunc("POST /start", func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			sendError(w, err)
			return
		}
		ioj := OfferJson{}
		json.Unmarshal(b, &ioj)
		log.Println("offer", ioj)
		b64Offer := wr.start(ioj.Offer)

		ooj := OfferJson{Offer: b64Offer}
		b, _ = json.Marshal(ooj)
		w.WriteHeader(http.StatusOK)
		w.Write(b)
	})

	mux.HandleFunc("POST /stop", func(w http.ResponseWriter, r *http.Request) {
		wr.stop()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	host := ":" + *port
	log.Printf("Serving on http://localhost%s\n", host)
	log.Fatal(http.ListenAndServe(host, mux))
}
