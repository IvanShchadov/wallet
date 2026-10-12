package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/IvanShchadov/wallet/internal/httpapi"
)

func main() {
	if err := run(); err != nil { // вся логика в run: она возвращает ошибку, а не падает внутри
		log.Fatal(err)
	}
}

func run() error {
	// ctx отменится по Ctrl+C или SIGTERM — так останавливают Docker и Kubernetes
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           httpapi.NewRouter(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Буфер 1 — правило из спринта 3: одна отправка может остаться без читателя,
	// например если сигнал пришёл раньше, — и горутина не зависнет
	errCh := make(chan error, 1)
	go func() { // ListenAndServe блокирует, пока сервер работает, поэтому он в горутине
		log.Printf("starting on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err // настоящая ошибка, например порт занят. ErrServerClosed — штатная остановка
		}
	}()

	select {
	case err := <-errCh: // сервер не смог стартовать — вернуть ошибку в main
		return fmt.Errorf("listen: %w", err)
	case <-ctx.Done(): // пришёл сигнал — идём останавливаться
	}

	log.Print("shutting down")

	// НОВЫЙ контекст: ctx от сигнала уже отменён, с ним Shutdown никого бы не дождался
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Shutdown перестаёт принимать новые запросы и ждёт текущие — не дольше 5 секунд
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}

	log.Print("server stopped")
	return nil
}
