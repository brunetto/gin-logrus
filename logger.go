package ginlogrus

import (
	"bytes"
	"math"
	"os"
	"time"
	"io/ioutil"
	"github.com/pkg/errors"

	"github.com/Sirupsen/logrus"
	"gopkg.in/gin-gonic/gin.v1"
	"gitlab.com/brunetto/hang"
)

// 2016-09-27 09:38:21.541541811 +0200 CEST
// 127.0.0.1 - frank [10/Oct/2000:13:55:36 -0700]
// "GET /apache_pb.gif HTTP/1.0" 200 2326
// "http://www.example.com/start.html"
// "Mozilla/4.08 [en] (Win98; I ;Nav)"

var timeFormat = "2006-01-02T15:04:05-0700"

// Logger is the logrus logger handler
func Logger(log *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// before handle actions
		var (
			requestBody []byte
			err error
		)

		// other handler can change c.Path so:
		path := c.Request.URL.Path
		start := time.Now()

		// Record request
		requestBody = hang.Tee(&(c.Request.Body))

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
