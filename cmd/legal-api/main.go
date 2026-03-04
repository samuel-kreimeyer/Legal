package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/samuel-kreimeyer/Legal/pkg/api"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	flag.Parse()

	handler := api.NewHandler().Routes()
	log.Printf("legal API listening on %s", *addr)
	log.Printf("docs: http://localhost%s/docs", *addr)
	log.Printf("openapi: http://localhost%s/openapi.json", *addr)
	if err := http.ListenAndServe(*addr, handler); err != nil {
		log.Fatal(err)
	}
}
