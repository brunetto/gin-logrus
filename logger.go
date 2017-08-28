package ginlogrus

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"math"
	"os"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	//"gopkg.in/gin-gonic/gin.v1"

)

// LogRequest activate the request body logging
var LogRequest = true

// LogResponse activate the response body logging
var LogResponse = true

// TimeFormat is the logs desired time format
var TimeFormat = "2006-01-02T15:04:05-0700"

// RespBodyExludedRoutes are the routes for which we don't want to log the response body
var RespBodyExludedRoutes = []string{"css", "font", "js", "assets", "icons", "img", "images", "script", "favicon.ico", "swagger"}

// Logger is the logrus logger handler
func Logger(log *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// before handle actions
		var (
			requestBody []byte
			//rbd string
			err error
		)

		// other handler can change c.Path so:
		path := c.Request.URL.Path
		start := time.Now()

		// Record request
		requestBody = Tee(&(c.Request.Body))

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

		// records few data
		entry := logrus.NewEntry(log).WithFields(logrus.Fields{
			"hostname":   hostname,
			"statusCode": statusCode,
			"latency":    latency, // time to process
			"clientIP":   clientIP,
			"method":     c.Request.Method,
			"path":       path,
			"referer":    referer,
			"dataLength": responseDataLength,
			"at":         c.Keys["at"],
		})

		// log the request body if desired
		if LogRequest && len(requestBody) > 0 {
			var data map[string]interface{}
			err = json.Unmarshal(requestBody, &data)
			if err != nil {
				entry = entry.WithField("requestBody", string(requestBody))
			}
			entry = entry.WithField("requestBody", data)
		} else {
			entry = entry.WithField("requestBody", "not logged or empty")
		}

		// log the response body if desired
		if LogResponse && !ExcludeRespBodyLog(path) {
			var data map[string]interface{}
			err = json.Unmarshal(responseBody.body.Bytes(), &data)
			if err != nil {
				entry = entry.WithField("responseBody", responseBody.body.String())
			}
			entry = entry.WithField("responseBody", data)
		} else {
			entry = entry.WithField("responseBody", "not logged or empty")
		}

		// Record errors
		if len(c.Errors) > 0 {
			entry.Error(c.Errors.ByType(gin.ErrorTypePrivate).String())
		} else {
			msg := ""
			if path == "/favicon.ico" {
				return
			}
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

// ExcludeRespBodyLog excludes some standard paths response body from being logged
func ExcludeRespBodyLog(path string) bool {
	for _, item := range RespBodyExludedRoutes {
		if strings.Contains(path, item) {
			return true
		}
	}
	return false
}

// RespBodyLogger contains the body of the response for logging purpose
type RespBodyLogger struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

// Write provides a copy of the response body to the logger
func (w RespBodyLogger) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// Tee provides a copy of the request body to be logged
func Tee(httpReqBody *io.ReadCloser) []byte {
	var b []byte
	b, _ = ioutil.ReadAll(*httpReqBody)
	*httpReqBody = ioutil.NopCloser(bytes.NewBuffer(b))
	return b
}
