package main

import (
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"log"
	"mrs/internal/api"
	"mrs/internal/config"
	"mrs/internal/service/audio"
	"mrs/internal/service/room"
	http_transport "mrs/internal/transport/http"
	ws_transport "mrs/internal/transport/ws"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	if _, err := os.Stat(".env"); err == nil {
		_ = godotenv.Load()
	}

	var cfg config.Config
	if err := envconfig.Process("", &cfg); err != nil {
		log.Fatal(err)
	}

	audioService, err := audio.NewServiceAudio(cfg.Youtube.Token, cfg.Youtube.Limit)
	if err != nil {
		log.Fatal(err)
	}

	cleanupInterval := 1 * time.Minute // раз в минуту проверять
	emptyRoomTTL := 5 * time.Minute

	roomService := room.NewServiceRoom(cleanupInterval, emptyRoomTTL)

	httpHandler := http_transport.NewHandler(audioService, roomService)
	wsHandler := ws_transport.NewWSHandler(roomService)

	a := api.NewAPI(
		api.Deps{
			WsHandler:   wsHandler,
			HttpHandler: httpHandler,
		})

	srv := &http.Server{
		Addr:              cfg.Rest.Address,
		Handler:           a,
		ReadTimeout:       time.Duration(cfg.Rest.ReadTimeout) * time.Second,
		WriteTimeout:      time.Duration(cfg.Rest.WriteTimeout) * time.Second,
		ReadHeaderTimeout: time.Duration(cfg.Rest.ReadHeaderTimeout) * time.Second,
		IdleTimeout:       time.Duration(cfg.Rest.IdleTimeout) * time.Second,
	}

	go func() {
		log.Printf("Listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	<-signalChan

	log.Println("Shutting down gracefully...")
}
