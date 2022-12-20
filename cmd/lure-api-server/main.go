package main

import (
	"flag"
	"net"
	"net/http"
	"os"

	"github.com/twitchtv/twirp"
	"go.arsenm.dev/logger"
	"go.arsenm.dev/logger/log"
	"go.arsenm.dev/lure/internal/api"
)

func init() {
	log.Logger = logger.NewPretty(os.Stderr)
}

func main() {
	addr := flag.String("a", ":8080", "Listen address for API server")
	logFile := flag.String("l", "", "Output file for JSON log")
	flag.Parse()

	if *logFile != "" {
		fl, err := os.Create(*logFile)
		if err != nil {
			log.Fatal("Error creating log file").Err(err).Send()
		}
		defer fl.Close()

		log.Logger = logger.NewMulti(log.Logger, logger.NewJSON(fl))
	}

	srv := api.NewAPIServer(
		lureWebAPI{db: gdb},
		twirp.WithServerPathPrefix(""),
	)

	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatal("Error starting listener").Err(err).Send()
	}

	log.Info("Starting HTTP API server").Str("addr", ln.Addr().String()).Send()
	
	err = http.Serve(ln, allowAllCORSHandler(srv))
	if err != nil {
		log.Fatal("Error while running server").Err(err).Send()
	}
}

func allowAllCORSHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.Header().Set("Access-Control-Allow-Origin", "*")
		res.Header().Set("Access-Control-Allow-Headers", "*")
		if req.Method == http.MethodOptions {
			return
		}
		h.ServeHTTP(res, req)
	})
}
