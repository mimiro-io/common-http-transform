package common_http_transform

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"github.com/DataDog/datadog-go/v5/statsd"
)

/******************************************************************************/

type Metrics interface {
	Incr(s string, tags []string, i int) TransformError
	Timing(s string, timed time.Duration, tags []string, i int) TransformError
	Gauge(s string, f float64, tags []string, i int) TransformError
}

type Logger interface {
	Error(message string, args ...any)
	Info(message string, args ...any)
	Debug(message string, args ...any)
	Warn(message string, args ...any)
	With(name string, value string) Logger
}

/******************************************************************************/

type StatsdMetrics struct {
	client statsd.ClientInterface
}

func (sm StatsdMetrics) Incr(name string, tags []string, rate int) TransformError {
	return Err(sm.client.Incr(name, tags, float64(rate)), LayerErrorInternal)
}

func (sm StatsdMetrics) Timing(name string, value time.Duration, tags []string, rate int) TransformError {
	return Err(sm.client.Timing(name, value, tags, float64(rate)), LayerErrorInternal)
}

func (sm StatsdMetrics) Gauge(name string, value float64, tags []string, rate int) TransformError {
	return Err(sm.client.Gauge(name, value, tags, float64(rate)), LayerErrorInternal)
}

func newMetrics(conf *Config) (Metrics, error) {
	var client statsd.ClientInterface
	if conf.LayerServiceConfig.StatsdEnabled {
		c, err := statsd.New(conf.LayerServiceConfig.StatsdAgentAddress,
			statsd.WithNamespace(conf.LayerServiceConfig.ServiceName),
			statsd.WithTags([]string{"application:" + conf.LayerServiceConfig.ServiceName}))
		if err != nil {
			return nil, err
		}

		client = c
	} else {
		client = &statsd.NoOpClient{}
	}

	return &StatsdMetrics{client: client}, nil
}

type logger struct {
	log zerolog.Logger
}

func (l *logger) With(name string, value string) Logger {
	subLogger := l.log.With().Str(name, value).Logger()
	return &logger{subLogger}
}

func (l *logger) Warn(message string, args ...any) {
	l.log.Warn().Fields(args).Msg(message)
}

func (l *logger) Error(message string, args ...any) {
	l.log.Error().Fields(args).Msg(message)
}

func (l *logger) Info(message string, args ...any) {
	l.log.Info().Fields(args).Msg(message)
}

func (l *logger) Debug(message string, args ...any) {
	l.log.Debug().Fields(args).Msg(message)
}

func NewLogger(serviceName string, format string, level string) Logger {
	var slevel zerolog.Level
	switch strings.ToLower(level) {
	case "debug":
		slevel = zerolog.DebugLevel
	case "info":
		slevel = zerolog.InfoLevel
	case "warn":
		slevel = zerolog.WarnLevel
	case "error":
		slevel = zerolog.ErrorLevel
	default:
		slevel = zerolog.InfoLevel
	}
	// Default level for this example is info, unless debug flag is present
	zerolog.SetGlobalLevel(slevel)
	zerolog.TimestampFieldName = "ts"
	zerolog.MessageFieldName = "msg"
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
	zerolog.CallerMarshalFunc = func(pc uintptr, file string, line int) string {
		pcs := make([]uintptr, 10)
		runtime.Callers(2, pcs)
		fs := runtime.CallersFrames(pcs)

		f, more := fs.Next()
		for more {
			if strings.HasPrefix(f.Function, "github.com/rs/zerolog") || strings.Contains(f.Function, "(*logger).") {
				f, more = fs.Next()
				continue
			}
			shortFile := filepath.Base(f.File)
			shortFunc := filepath.Base(f.Function)
			return shortFunc + " (" + shortFile + ":" + strconv.Itoa(f.Line) + ")"
		}
		return file + ":" + strconv.Itoa(line)
	}

	base := zerolog.New(os.Stdout)
	if format == "text" {
		base = base.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})
	}
	log := base.With().
		Timestamp().
		Caller().
		Str("go.version", runtime.Version()).
		Str("service", serviceName).
		Logger()

	return &logger{log}
}
