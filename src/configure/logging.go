package configure

import "github.com/sirupsen/logrus"

func InitLogging(level string) {
	logrus.SetReportCaller(true)
	logrus.SetFormatter(&logrus.JSONFormatter{})
	if lvl, err := logrus.ParseLevel(level); err == nil {
		logrus.SetLevel(lvl)
	}
}
