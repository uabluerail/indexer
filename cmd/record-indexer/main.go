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

	"github.com/uabluerail/indexer/util/gormzerolog"
)

type Config struct {
	LogFile             string
	LogFormat           string   `default:"text"`
	LogLevel            int64    `default:"1"`
	MetricsPort         string   `split_words:"true"`
	DBUrl               string   `envconfig:"POSTGRES_URL"`
	Workers             int      `default:"2"`
	CollectionBlacklist []string `split_words:"true"`
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

	limiter, err := NewLimiter(db)
	if err != nil {
		return fmt.Errorf("failed to create limiter: %w", err)
	}

	ch := make(chan WorkItem)
	pool := NewWorkerPool(ch, db, config.Workers, limiter)
	if err := pool.Start(ctx); err != nil {
		return fmt.Errorf("failed to start worker pool: %w", err)
	}
	pool.BlacklistCollections(config.CollectionBlacklist)

	scheduler := NewScheduler(ch, db)
	if err := scheduler.Start(ctx); err != nil {
		return fmt.Errorf("failed to start scheduler: %w", err)
	}

	log.Info().Msgf("Starting HTTP listener on %q...", config.MetricsPort)
	AddAdminHandlers(limiter, pool)
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
	return <-errCh
}

func main() {
	flag.StringVar(&config.LogFile, "log", "", "Path to the log file. If empty, will log to stderr")
	flag.StringVar(&config.LogFormat, "log-format", "text", "Logging format. 'text' or 'json'")
	flag.Int64Var(&config.LogLevel, "log-level", 1, "Log level. -1 - trace, 0 - debug, 1 - info, 5 - panic")
	flag.IntVar(&config.Workers, "workers", 2, "Number of workers to start with")

	if err := envconfig.Process("indexer", &config); err != nil {
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
