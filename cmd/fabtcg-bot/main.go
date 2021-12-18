package main

import (
	"context"
	"fmt"
	"github.com/alecthomas/kong"
	"github.com/cbrgm/fabtcg-bot/fabdb"
	"github.com/cbrgm/fabtcg-bot/metrics"
	"github.com/cbrgm/fabtcg-bot/telegram"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/oklog/run"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

const (
	levelDebug = "debug"
	levelInfo  = "info"
	levelWarn  = "warn"
	levelError = "error"
)

var (
	// Version of fabtcg-bot.
	Version string
	// Revision or Commit this binary was built from.
	Revision string
	// GoVersion running this binary.
	GoVersion = runtime.Version()
	// StartTime has the time this was started.
	StartTime = time.Now()
)

var cli struct {
	HttpAddr string `name:"http.addr" default:"0.0.0.0:8080" help:"The address the fabtcg-bot metrics are exposed"`
	LogLevel string `name:"log.level" default:"info" enum:"error,warn,info,debug" help:"The log level to use for filtering logs"`

	cliTelegram
	cliMetrics
}

type cliMetrics struct {
	EnableProfiling      bool   `name:"metrics.profile" default:"true" help:"Enable pprof profiling"`
	EnableRuntimeMetrics bool   `name:"metrics.runtime" default:"true" help:"Enable bot runtime metrics"`
	EnableMetrics        bool   `name:"metrics.enabled" default:"true" help:"Enable bot metrics"`
	MetricsPrefix        string `name:"metrics.prefix" default:"" help:"Set metrics prefix path"`
}

type cliTelegram struct {
	Admins []int  `name:"telegram.admin" help:"The ID of the initial Telegram Admin"`
	Token  string `required:"true" name:"telegram.token" env:"TELEGRAM_TOKEN" help:"The token used to connect with Telegram"`
}

func main() {
	_ = kong.Parse(&cli,
		kong.Name("fabtcg-bot"),
	)

	levelFilter := map[string]level.Option{
		levelError: level.AllowError(),
		levelWarn:  level.AllowWarn(),
		levelInfo:  level.AllowInfo(),
		levelDebug: level.AllowDebug(),
	}

	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = level.NewFilter(logger, levelFilter[cli.LogLevel])
	logger = log.With(logger,
		"ts", log.DefaultTimestampUTC,
		"caller", log.DefaultCaller,
	)

	metricOptions := metrics.Options{
		Enabled:              cli.EnableMetrics,
		Prefix:               cli.MetricsPrefix,
		EnableProfile:        cli.EnableProfiling,
		EnableRuntimeMetrics: cli.EnableRuntimeMetrics,
	}

	prom := metrics.NewPrometheus(metricOptions)
	ctx, cancel := context.WithCancel(context.Background())

	var gr run.Group
	{
		tlogger := log.With(logger, "component", "telegram")

		token := cli.Token
		allowlist := cli.Admins
		client := fabdb.NewFabDBClient()

		bot, err := telegram.NewBot(client, token,
			telegram.WithLogger(tlogger),
			telegram.WithMetrics(prom),
			telegram.WithAllowlist(allowlist...),
			telegram.WithStartTime(StartTime),
			telegram.WithRevision(Revision),
		)
		if err != nil {
			level.Error(tlogger).Log("msg", "failed to initialize telegram bot", "err", err)
			os.Exit(2)
		}

		gr.Add(func() error {
			level.Info(tlogger).Log(
				"msg", "starting fabtcg bot",
				"version", Version,
				"revision", Revision,
				"goVersion", GoVersion,
			)
			return bot.Run(ctx)
		}, func(err error) {
			cancel()
		})
	}
	{
		wlogger := log.With(logger, "component", "webserver")
		handleHealth := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}

		m := http.NewServeMux()
		if metricOptions.Enabled {
			m.Handle("/metrics", metrics.HandlerFor(prom, metricOptions))
		}
		m.HandleFunc("/health", handleHealth)
		m.HandleFunc("/healthz", handleHealth)

		s := http.Server{
			Addr:    cli.HttpAddr,
			Handler: m,
		}

		gr.Add(func() error {
			level.Info(wlogger).Log("msg", "starting webserver", "addr", cli.HttpAddr)
			return s.ListenAndServe()
		}, func(err error) {
			_ = s.Shutdown(context.Background())
		})
	}
	{
		sig := make(chan os.Signal)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

		gr.Add(func() error {
			<-sig
			return nil
		}, func(err error) {
			cancel()
			close(sig)
		})
	}

	if err := gr.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
