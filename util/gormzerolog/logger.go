package gormzerolog

import (
	"context"
	"errors"
	"time"

	"github.com/rs/zerolog"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type logAdapter struct {
	logger *zerolog.Logger
	config logger.Config
}

// New returns a logger that will use zerolog to log messages.
// Both `config` and `logger` are optional. If `logger` is `nil`,
// it will be retrieved from context.Context. Config options that
// are configured in zerolog are not supported (currently that's
// `Colorful` and `LogLevel`).
func New(config *logger.Config, logger *zerolog.Logger) logger.Interface {
	r := &logAdapter{logger: logger}
	if config != nil {
		r.config = *config
	}
	return r
}

func (l *logAdapter) ctx(ctx context.Context) *zerolog.Logger {
	if l.logger != nil {
		return l.logger
	}
	return zerolog.Ctx(ctx)
}

// Stuff below was copy-pasted from https://github.com/go-gorm/gorm/blob/v1.25.7/logger/logger.go
// and edited.

func (l *logAdapter) LogMode(level logger.LogLevel) logger.Interface {
	return l
}

// Info print info
func (l *logAdapter) Info(ctx context.Context, msg string, data ...interface{}) {
	l.ctx(ctx).Info().CallerSkipFrame(3).Msgf(msg, data...)
}

// Warn print warn messages
func (l *logAdapter) Warn(ctx context.Context, msg string, data ...interface{}) {
	l.ctx(ctx).Warn().CallerSkipFrame(3).Msgf(msg, data...)
}

// Error print error messages
func (l *logAdapter) Error(ctx context.Context, msg string, data ...interface{}) {
	l.ctx(ctx).Error().CallerSkipFrame(3).Msgf(msg, data...)
}

// Trace print sql message
//
//nolint:cyclop
func (l *logAdapter) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()
	switch {
	case err != nil && (!errors.Is(err, gorm.ErrRecordNotFound) || !l.config.IgnoreRecordNotFoundError):
		log := l.ctx(ctx).Error().CallerSkipFrame(3).Err(err)
		if rows >= 0 {
			log.Int64("rows", rows)
		}
		log.Dur("dur", elapsed)
		log.Msgf("%s: %s", sql, err)
	case elapsed > l.config.SlowThreshold && l.config.SlowThreshold != 0:
		log := l.ctx(ctx).Warn().CallerSkipFrame(3)
		if rows >= 0 {
			log.Int64("rows", rows)
		}
		log.Dur("dur", elapsed)
		log.Msgf("SLOW SQL: %s", sql)
	default:
		log := l.ctx(ctx).Trace().CallerSkipFrame(3)
		if rows >= 0 {
			log.Int64("rows", rows)
		}
		log.Dur("dur", elapsed)
		log.Msgf("%s", sql)
	}
}

// ParamsFilter filter params
func (l *logAdapter) ParamsFilter(ctx context.Context, sql string, params ...interface{}) (string, []interface{}) {
	if l.config.ParameterizedQueries {
		return sql, nil
	}
	return sql, params
}
