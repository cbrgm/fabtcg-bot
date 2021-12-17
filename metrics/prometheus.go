package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"strings"
	"time"
)

const (
	promNamespace         = "fabtcgbot"
	promTelegramSubsystem = "telegram"
)

const (
	TelegramMessageEventType     = "message"
	TelegramInlineQueryEventType = "inline"
)

// Prometheus implements the prometheus metrics backend.
type Prometheus struct {
	telegramCommandsM       *prometheus.CounterVec
	telegramEventsIncomingM *prometheus.CounterVec
	telegramEventsOutgoingM *prometheus.CounterVec
	opts                    Options
	registry                *prometheus.Registry
	handler                 http.Handler
}

func NewDefaultPrometheus() *Prometheus {
	return NewPrometheus(DefaultOptions())
}

// NewPrometheus returns a new Prometheus metric backend.
func NewPrometheus(opts Options) *Prometheus {
	namespace := promNamespace
	if opts.Prefix != "" {
		namespace = strings.TrimSuffix(opts.Prefix, ".")
	}

	telegramCommands := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: promTelegramSubsystem,
		Name:      "commands_total",
		Help:      "Total number of command requests.",
	}, []string{"command"})

	telegramEventsIncoming := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: promTelegramSubsystem,
		Name:      "events_incoming_total",
		Help:      "Total number of incoming messages.",
	}, []string{"type"})

	telegramEventsOutgoing := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: promTelegramSubsystem,
		Name:      "events_outgoing_total",
		Help:      "Total number of outgoing messages.",
	}, []string{"type"})

	p := &Prometheus{
		telegramCommandsM:       telegramCommands,
		telegramEventsIncomingM: telegramEventsIncoming,
		telegramEventsOutgoingM: telegramEventsOutgoing,
		opts:                    opts,
		registry:                opts.PrometheusRegistry,
		handler:                 nil,
	}

	if p.registry == nil {
		p.registry = prometheus.NewRegistry()
	}
	p.registerMetrics()
	return p
}

func (p *Prometheus) registerMetrics() {
	p.registry.MustRegister(p.telegramCommandsM)
	p.registry.MustRegister(p.telegramEventsIncomingM)
	p.registry.MustRegister(p.telegramEventsOutgoingM)

	if p.opts.EnableRuntimeMetrics {
		p.registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
		p.registry.MustRegister(prometheus.NewGoCollector())
	}
}

func (p *Prometheus) CreateHandler() http.Handler {
	return promhttp.HandlerFor(p.registry, promhttp.HandlerOpts{})
}

func (p *Prometheus) getHandler() http.Handler {
	if p.handler != nil {
		return p.handler
	}
	p.handler = p.CreateHandler()
	return p.handler
}

// RegisterHandler satisfies Metrics interface.
func (p *Prometheus) RegisterHandler(path string, mux *http.ServeMux) {
	promHandler := p.getHandler()
	mux.Handle(path, promHandler)
}

// sinceStart returns the seconds passed since the start time until now.
func (p *Prometheus) sinceStart(start time.Time) float64 {
	return time.Since(start).Seconds()
}

func (p *Prometheus) IncTelegramCommands(cmd string) {
	p.telegramCommandsM.WithLabelValues(cmd).Inc()
}

func (p *Prometheus) IncTelegramEventsIncoming(eventType string) {
	p.telegramEventsIncomingM.WithLabelValues(eventType).Inc()
}

func (p *Prometheus) IncTelegramEventsOutgoing(eventType string) {
	p.telegramEventsOutgoingM.WithLabelValues(eventType).Inc()
}
