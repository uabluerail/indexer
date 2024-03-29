package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	_ "github.com/joho/godotenv/autoload"
	"github.com/kelseyhightower/envconfig"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/uabluerail/indexer/pds"
	"github.com/uabluerail/indexer/util/gormzerolog"
)

type Config struct {
	LogFile     string
	LogFormat   string `default:"text"`
	LogLevel    int64  `default:"1"`
	MetricsPort string `split_words:"true"`
	DBUrl       string `envconfig:"POSTGRES_URL"`
	Relays      string
}

var config Config

func runMain(ctx context.Context) error {
	ctx = setupLogging(ctx)
	log := zerolog.Ctx(ctx)
	log.Debug().Msgf("Starting up...")
	db, err := gorm.Open(postgres.Open(config.DBUrl), &gorm.Config{
		Logger: gormzerolog.New(&logger.Config{
			SlowThreshold:             3 * time.Second,
			IgnoreRecordNotFoundError: true,
		}, nil),
	})
	if err != nil {
		return fmt.Errorf("connecting to the database: %w", err)
	}
	log.Debug().Msgf("DB connection established")

	if config.Relays != "" {
		for _, host := range strings.Split(config.Relays, ",") {
			c, err := NewRelayConsumer(ctx, host, db)
			if err != nil {
				log.Error().Err(err).Msgf("Failed to create relay consumer for %q: %s", host, err)
			}
			c.Start(ctx)
		}
	}

	consumersCh := make(chan struct{})
	go runConsumers(ctx, db, consumersCh)

	log.Info().Msgf("Starting HTTP listener on %q...", config.MetricsPort)
	http.Handle("/metrics", promhttp.Handler())
	srv := &http.Server{Addr: fmt.Sprintf(":%s", config.MetricsPort)}
	errCh := make(chan error)
	go func() {
		errCh <- srv.ListenAndServe()
	}()
	select {
	case <-ctx.Done():
		if err := srv.Shutdown(context.Background()); err != nil {
			return fmt.Errorf("HTTP server shutdown failed: %w", err)
		}
	}
	log.Info().Msgf("Waiting for consumers to stop...")
	<-consumersCh
	return <-errCh
}

func runConsumers(ctx context.Context, db *gorm.DB, doneCh chan struct{}) {
	log := zerolog.Ctx(ctx)
	defer close(doneCh)

	type consumerHandle struct {
		cancel   context.CancelFunc
		consumer *Consumer
	}

	running := map[string]consumerHandle{}

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	t := make(chan time.Time, 1)
	t <- time.Now()

	for {
		select {
		case <-t:
			remotes := []pds.PDS{}
			if err := db.Find(&remotes).Error; err != nil {
				log.Error().Err(err).Msgf("Failed to get a list of known PDSs: %s", err)
				break
			}

			shouldBeRunning := map[string]pds.PDS{}
			for _, remote := range remotes {
				if remote.Disabled {
					continue
				}
				shouldBeRunning[remote.Host] = remote
			}

			for host, handle := range running {
				if _, found := shouldBeRunning[host]; found {
					continue
				}
				handle.cancel()
				_ = handle.consumer.Wait(ctx)
				delete(running, host)
			}

			for host, remote := range shouldBeRunning {
				if _, found := running[host]; found {
					continue
				}
				subCtx, cancel := context.WithCancel(ctx)

				c, err := NewConsumer(subCtx, &remote, db)
				if err != nil {
					log.Error().Err(err).Msgf("Failed to create a consumer for %q: %s", remote.Host, err)
					cancel()
					continue
				}
				if err := c.Start(subCtx); err != nil {
					log.Error().Err(err).Msgf("Failed ot start a consumer for %q: %s", remote.Host, err)
					cancel()
					continue
				}

				running[host] = consumerHandle{
					cancel:   cancel,
					consumer: c,
				}
			}

		case <-ctx.Done():
			for host, handle := range running {
				handle.cancel()
				_ = handle.consumer.Wait(ctx)
				delete(running, host)
			}

		case v := <-ticker.C:
			// Non-blocking send.
			select {
			case t <- v:
			default:
			}
		}
	}
}

func main() {
	flag.StringVar(&config.LogFile, "log", "", "Path to the log file. If empty, will log to stderr")
	flag.StringVar(&config.LogFormat, "log-format", "text", "Logging format. 'text' or 'json'")
	flag.Int64Var(&config.LogLevel, "log-level", 1, "Log level. -1 - trace, 0 - debug, 1 - info, 5 - panic")
	flag.StringVar(&config.Relays, "relays", "", "List of relays to connect to (for discovering new PDSs)")

	if err := envconfig.Process("consumer", &config); err != nil {
		log.Fatalf("envconfig.Process: %s", err)
	}

	flag.Parse()

	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	if err := runMain(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func setupLogging(ctx context.Context) context.Context {
	logFile := os.Stderr

	if config.LogFile != "" {
		f, err := os.OpenFile(config.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalf("Failed to open the specified log file %q: %s", config.LogFile, err)
		}
		logFile = f
	}

	var output io.Writer

	switch config.LogFormat {
	case "json":
		output = logFile
	case "text":
		prefixList := []string{}
		info, ok := debug.ReadBuildInfo()
		if ok {
			prefixList = append(prefixList, info.Path+"/")
		}

		basedir := ""
		_, sourceFile, _, ok := runtime.Caller(0)
		if ok {
			basedir = filepath.Dir(sourceFile)
		}

		if basedir != "" && strings.HasPrefix(basedir, "/") {
			prefixList = append(prefixList, basedir+"/")
			head, _ := filepath.Split(basedir)
			for head != "/" {
				prefixList = append(prefixList, head)
				head, _ = filepath.Split(strings.TrimSuffix(head, "/"))
			}
		}

		output = zerolog.ConsoleWriter{
			Out:        logFile,
			NoColor:    true,
			TimeFormat: time.RFC3339,
			PartsOrder: []string{
				zerolog.LevelFieldName,
				zerolog.TimestampFieldName,
				zerolog.CallerFieldName,
				zerolog.MessageFieldName,
			},
			FormatFieldName:  func(i interface{}) string { return fmt.Sprintf("%s:", i) },
			FormatFieldValue: func(i interface{}) string { return fmt.Sprintf("%s", i) },
			FormatCaller: func(i interface{}) string {
				s := i.(string)
				for _, p := range prefixList {
					s = strings.TrimPrefix(s, p)
				}
				return s
			},
		}
	default:
		log.Fatalf("Invalid log format specified: %q", config.LogFormat)
	}

	logger := zerolog.New(output).Level(zerolog.Level(config.LogLevel)).With().Caller().Timestamp().Logger()

	ctx = logger.WithContext(ctx)

	zerolog.DefaultContextLogger = &logger
	log.SetOutput(logger)

	return ctx
}
