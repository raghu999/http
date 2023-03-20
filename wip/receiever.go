package httplogreceiver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
  // "go.opentelemetry.io/collector/pdata"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
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
		format := r.URL.Query().Get("format")
		if format == "" {
			format = "json"
		}
		var logs plog.Logs
		switch format {
		case "json":
			body, err := decodeJSON(r)
			if err != nil {
				http.Error(w, fmt.Sprintf("failed to decode JSON: %v", err), http.StatusBadRequest)
				return
			}
			logs = body
		case "raw":
			body, err := encodeRaw(r)
			if err != nil {
				http.Error(w, fmt.Sprintf("failed to encode raw: %v", err), http.StatusBadRequest)
				return
			}
			logs = body
		default:
			http.Error(w, fmt.Sprintf("invalid format specified: %s", format), http.StatusBadRequest)
			return
		}

		h.createTimestamp(logs.ResourceLogs().At(0).InstrumentationLibraryLogs().At(0).Logs())
		h.consumer.ConsumeLogs(context.Background(), logs)

		fmt.Fprintf(w, "Successfully processed logs\n")
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

func (h *httplogReceiver) createTimestamp(logs plog.Logs) {
	now := plogs.TimestampFromTime(time.Now())
	for i := 0; i < logs.Len(); i++ {
		logRecord := logs.At(i)
		if !logRecord.Timestamp().Valid() {
			logRecord.SetTimestamp(now)
		}
	}
}

func decodeJSON(body []byte) (plog.Logs, error) {
	var jsonLogs []map[string]interface{}
	err := json.Unmarshal(body, &jsonLogs)
	if err != nil {
		return plog.Logs{}, err
	}

	logs := plog.NewLogs()
	rls := logs.ResourceLogs().AppendEmpty()
	ill := rls.InstrumentationLibraryLogs().AppendEmpty()

	for _, jl := range jsonLogs {
		ts, ok := jl["timestamp"].(string)
		if !ok {
			ts = time.Now().UTC().Format(time.RFC3339Nano)
		}

		tsNano, err := time.Parse(time.RFC3339Nano, ts)
		if err != nil {
			return plog.Logs{}, err
		}

		logRecord := plog.NewLogRecord()
		logRecord.SetTimestamp(pcommon.NewTimestampFromTime(tsNano))

		for k, v := range jl {
			if k != "timestamp" {
				attr := pcommon.NewAttributeKeyValue(k, v)
				logRecord.Attributes().Insert(attr)
			}
		}

		ill.Logs().Append(logRecord)
	}

	return logs, nil
}
