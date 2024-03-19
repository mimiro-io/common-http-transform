package common_http_transform

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	egdm "github.com/mimiro-io/entity-graph-data-model"
)

type transformWebService struct {
	// service specific service core
	transformService TransformService
	e                *echo.Echo
	metrics          Metrics
	logger           Logger
	config           *Config
}

func newTransformService(config *Config, logger Logger, metrics Metrics, transformService TransformService) (*transformWebService, error) {
	e := echo.New()
	e.HideBanner = true
	mw(logger, metrics, e)
	s := &transformWebService{config: config, logger: logger, metrics: metrics, transformService: transformService, e: e}
	e.GET("/health", s.health)
	e.POST("/transform", s.transform)
	return s, nil
}

// wrap all handlers with middleware
func mw(logger Logger, metrics Metrics, e *echo.Echo) {
	skipper := func(c echo.Context) bool {
		// skip health check
		return strings.HasPrefix(c.Request().URL.Path, "/health")
	}
	defaultErrorHandler := e.HTTPErrorHandler
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		if c.Response().Committed {
			logger.Error("Internal Error. Response already committed to 200 but will produce truncated/invalid payload.", "error", err.Error())
		} else {
			logger.Error("Internal Error", "error", err.Error())
		}
		defaultErrorHandler(err, c)
	}
	e.Use(
		// Request logging and HTTP metrics
		func(next echo.HandlerFunc) echo.HandlerFunc {
			// service := core.Config.SystemConfig.ServiceName()
			return func(c echo.Context) error {
				if skipper(c) {
					return next(c)
				}

				start := time.Now()
				tags := []string{
					// fmt.Sprintf("application:%s", service),
					fmt.Sprintf("method:%s", strings.ToLower(c.Request().Method)),
					fmt.Sprintf("url:%s", strings.ToLower(c.Request().RequestURI)),
					fmt.Sprintf("status:%d", c.Response().Status),
				}

				// Recover from panic
				defer func() {
					if r := recover(); r != nil {
						err, ok := r.(error)
						if !ok {
							err = fmt.Errorf("%v", r)
						}
						stack := make([]byte, middleware.DefaultRecoverConfig.StackSize)
						length := runtime.Stack(stack, !middleware.DefaultRecoverConfig.DisableStackAll)
						if !middleware.DefaultRecoverConfig.DisablePrintStack {
							msg := fmt.Sprintf("[PANIC RECOVER] %v %s\n", err, stack[:length])
							logger.Warn(msg)
						}
						c.Error(err)
					}
				}()

				// next middleware/handler
				err := next(c)
				if err != nil {
					c.Error(err)
				}

				timed := time.Since(start)

				err = metrics.Incr("http.count", tags, 1)
				err = metrics.Timing("http.time", timed, tags, 1)
				err = metrics.Gauge("http.size", float64(c.Response().Size), tags, 1)
				if err != nil {
					logger.Warn("Error with metrics", "error", err.Error())
				}

				msg := fmt.Sprintf("%d - %s %s (time: %s, size: %d, user_agent: %s)",
					c.Response().Status, c.Request().Method, c.Request().RequestURI, timed.String(),
					c.Response().Size, c.Request().UserAgent())

				args := []any{
					"time", timed.String(),
					"request", fmt.Sprintf("%s %s", c.Request().Method, c.Request().RequestURI),
					"status", c.Response().Status,
					"size", c.Response().Size,
					"user_agent", c.Request().UserAgent(),
				}

				id := c.Request().Header.Get(echo.HeaderXRequestID)
				if id == "" {
					id = c.Response().Header().Get(echo.HeaderXRequestID)
					args = append(args, "request_id", id)
				}

				logger.Info(msg, args...)

				return nil
			}
		})
}

func (ws *transformWebService) Start() error {
	port := ws.config.LayerServiceConfig.Port
	ws.logger.Info(fmt.Sprintf("Starting Http server on :%s", port))
	go func() {
		_ = ws.e.Start(":" + port.String())
	}()

	return nil
}

func (ws *transformWebService) Stop(ctx context.Context) error {
	return ws.e.Shutdown(ctx)
}

func (ws *transformWebService) health(c echo.Context) error {
	return c.String(http.StatusOK, "running")
}

func (ws *transformWebService) transform(c echo.Context) error {
	parser := egdm.NewEntityParser(egdm.NewNamespaceContext())
	parser.WithNoContext().WithExpandURIs()
	ec, err := parser.LoadEntityCollection(c.Request().Body)

	if err != nil {
		ws.logger.Warn(err.Error())
		return echo.NewHTTPError(http.StatusBadRequest, "could not parse the request body: %s", err.Error())
	}

	transformed, err := ws.transformService.Transform(ec)
	if err != nil {
		ws.logger.Warn(err.Error())
		return echo.NewHTTPError(http.StatusInternalServerError, "could not process the entities: %", err.Error())
	}

	c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
	c.Response().WriteHeader(http.StatusOK)

	transformed.SetOmitContextOnWrite(true)
	err = transformed.WriteEntityGraphJSON(c.Response().Writer)
	if err != nil {
		ws.logger.Warn(err.Error())
		return echo.NewHTTPError(http.StatusInternalServerError, "could not write the response: %s", err.Error())
	}
	c.Response().Flush()

	return nil
}
