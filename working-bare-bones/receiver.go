package httplogreceiver

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
	"go.uber.org/zap"

	"github.com/gorilla/mux"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
)

type httplogReceiver struct {
	config   Config
	consumer consumer.Logs
	server   *http.Server
}

// NewReceiver creates a new httplogReceiver reference.
func NewReceiver(set receiver.CreateSettings, config *Config, nextConsumer consumer.Logs) (receiver.Logs, error) {
	lis, err := net.Listen("tcp", config.Endpoint)
	if err != nil {
		return nil, err
	}

	r := mux.NewRouter()

	// Register the handler function for incoming HTTP POST requests.
	r.HandleFunc("/logs", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Received a POST request\n")
	})

	srv := &http.Server{
		ReadHeaderTimeout: time.Duration(5) * time.Second,
		WriteTimeout:      time.Duration(10) * time.Second,
		Handler:           r,
	}

	go func() {
		if err := srv.Serve(lis); err != http.ErrServerClosed {
			set.Logger.Error("httplogreceiver serve error: %v", zap.Error(err))
		}
	}()

	return &httplogReceiver{
		config:   *config,
		consumer: nextConsumer,
		server:   srv,
	}, nil
}

func (h *httplogReceiver) Start(_ context.Context, host component.Host) error {
	host.ReportFatalError(h.server.ListenAndServe())
	return nil
}

func (h *httplogReceiver) Shutdown(context.Context) error {
	return h.server.Shutdown(context.Background())
}
