package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"github.com/gocql/gocql"
	_ "github.com/joho/godotenv/autoload"
	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog"
	"github.com/scylladb/gocqlx/v3"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/uabluerail/indexer/pds"
	"github.com/uabluerail/indexer/repo"
	"github.com/uabluerail/indexer/util/gormzerolog"
)

type Config struct {
	LogFile      string
	LogFormat    string `default:"text"`
	LogLevel     int64  `default:"1"`
	DBUrl        string `envconfig:"POSTGRES_URL"`
	ScyllaDBAddr string `envconfig:"SCYLLADB_ADDR"`
}

var config Config

func runMain(ctx context.Context) error {
	ctx = setupLogging(ctx)
	log := zerolog.Ctx(ctx)
	log.Debug().Msgf("Starting up...")
	db, err := gorm.Open(postgres.Open(config.DBUrl), &gorm.Config{
		Logger: gormzerolog.New(&logger.Config{
			SlowThreshold:             1 * time.Second,
			IgnoreRecordNotFoundError: true,
		}, nil),
	})
	if err != nil {
		return fmt.Errorf("connecting to the database: %w", err)
	}
	log.Debug().Msgf("DB connection established")

	for _, f := range []func(*gorm.DB) error{
		pds.AutoMigrate,
		repo.AutoMigrate,
	} {
		if err := f(db); err != nil {
			return fmt.Errorf("auto-migrating DB schema: %w", err)
		}
	}

	if config.ScyllaDBAddr != "" {
		scylla := gocql.NewCluster(config.ScyllaDBAddr)
		session, err := gocqlx.WrapSession(scylla.CreateSession())
		if err != nil {
			return fmt.Errorf("Creating ScyllaDB session: %w", err)
		}

		err = session.ExecStmt(
			`CREATE KEYSPACE IF NOT EXISTS bluesky WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1}`)
		if err != nil {
			return fmt.Errorf("Creating keyspace: %w", err)
		}

		err = session.ExecStmt(`CREATE TABLE IF NOT EXISTS bluesky.records (
			repo text,
			collection text,
			rkey text,
			created_at timestamp,
			at_rev text,
			deleted boolean,
			record text,
			PRIMARY KEY ((repo, collection), rkey, at_rev)
		) WITH CLUSTERING ORDER BY (rkey ASC, at_rev DESC)`)
		if err != nil {
			return fmt.Errorf("Creating records table: %w", err)
		}
	}

	log.Debug().Msgf("DB schema updated")
	return nil
}

func main() {
	flag.StringVar(&config.LogFile, "log", "", "Path to the log file. If empty, will log to stderr")
	flag.StringVar(&config.LogFormat, "log-format", "text", "Logging format. 'text' or 'json'")
	flag.Int64Var(&config.LogLevel, "log-level", 1, "Log level. -1 - trace, 0 - debug, 1 - info, 5 - panic")

	if err := envconfig.Process("update-db-schema", &config); err != nil {
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
