package log

import (
	"github.com/sirupsen/logrus"
)

var Log = logrus.StandardLogger()

func init() {
	Log.ReportCaller = true
}
