package utils

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// StartSvrGracefulShutdown starts the server with graceful shutdown
func StartSvrGracefulShutdown(srv *http.Server) {

	//running server in bg
	go func() {
		log.Println("Svr running at", srv.Addr)
		if err := srv.ListenAndServe(); err != nil {
			log.Fatalf("Listen&Serve error: %v\n", err)
		}
	}()

	//catch interruptions
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Println("Shutting this server down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Forced svr shutdown: %v\n", err)
	}
	log.Println("Svr exiting")

}
