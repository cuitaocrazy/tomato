package logger

import (
	"github.com/sirupsen/logrus"
)

var loggerMap = make(map[string]*logrus.Entry)

func init() {
	logrus.SetLevel(logrus.TraceLevel)
}
func GetLogger(moduleName string) *logrus.Entry {
	if _, ok := loggerMap[moduleName]; !ok {
		logger := logrus.WithFields(logrus.Fields{
			"module": moduleName,
		})
		loggerMap[moduleName] = logger
	}
	return loggerMap[moduleName]
}
