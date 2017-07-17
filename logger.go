package ginlogrus

import (
	"math"
	"os"
	"time"

	"github.com/Sirupsen/logrus"
	"gopkg.in/gin-gonic/gin.v1"
	"gitlab.com/brunetto/hang"
	"bytes"
	"strings"
)

// 2016-09-27 09:38:21.541541811 +0200 CEST
// 127.0.0.1 - frank [10/Oct/2000:13:55:36 -0700]
// "GET /apache_pb.gif HTTP/1.0" 200 2326
// "http://www.example.com/start.html"
// "Mozilla/4.08 [en] (Win98; I ;Nav)"

var TimeFormat = "2006-01-02T15:04:05-0700"

var RespBodySizeLimit int = 5000

var RespBodyExludedRoutes = []string{"css", "font", "js", "assets", "icons", "img", "images", "script"}

func ExcludeRespBodyLog(path string) bool {
	for _, item := range RespBodyExludedRoutes {
		if strings.Contains(path, item) {
			return true
		}
	}
	return false
}

type RespBodyLogger struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w RespBodyLogger) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// Logger is the logrus logger handler
func Logger(log *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// before handle actions
		var (
			requestBody []byte
			rbd string
			err error
		)

		// other handler can change c.Path so:
		path := c.Request.URL.Path
		start := time.Now()

		// Record request
		requestBody = hang.Tee(&(c.Request.Body))

		// Record response
		responseBody := &RespBodyLogger{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = responseBody

		c.Next() // here handlers are executed

		// after handle actions
		stop := time.Since(start)
		latency := int(math.Ceil(float64(stop.Nanoseconds()) / 1000.0))
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		//clientUserAgent := c.Request.UserAgent()
		referer := c.Request.Referer()
		hostname, err := os.Hostname()
		if err != nil {
			hostname = "unknow"
		}
		responseDataLength := c.Writer.Size()
		if responseDataLength < 0 {
			responseDataLength = 0
		}

		if !ExcludeRespBodyLog(path) {
			if responseDataLength < RespBodySizeLimit {
				rbd = responseBody.body.String()
			} else {
				if responseBody.body.Len() > 0 {
					responseBody.body.Truncate(RespBodySizeLimit)
				} else {
					rbd = "... negative size body... "
				}
				rbd = responseBody.body.String() + "... [truncated]"
			}
		} else {
			rbd = "not logged"
		}

		entry := logrus.NewEntry(log).WithFields(logrus.Fields{
			"hostname":   hostname,
			"statusCode": statusCode,
			"latency":    latency, // time to process
			"clientIP":   clientIP,
			"method":     c.Request.Method,
			"path":       path,
			"referer":    referer,
			"dataLength": responseDataLength,
			"at": c.Keys["at"],
			"requestBody": string(requestBody),
			//"userAgent":  clientUserAgent,
			"responseBody": rbd,
		})



		if len(c.Errors) > 0 {
			entry.Error(c.Errors.ByType(gin.ErrorTypePrivate).String())
		} else {
			msg := ""
			if path == "/favicon.ico" {return}
			if statusCode > 499 {
				entry.Error(msg)
			} else if statusCode > 399 {
				entry.Warn(msg)
			} else {
				entry.Info(msg)
			}
		}
	}
}
