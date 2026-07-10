package logger

import (
	"fmt"
	"os"
	"runtime"

	"github.com/sirupsen/logrus"
)

var Log *logrus.Logger

func init() {
	Log = logrus.New()
	Log.SetReportCaller(true)
	Log.SetFormatter(&logrus.TextFormatter{
		ForceColors:            true,
		DisableTimestamp:       false,
		DisableLevelTruncation: false,
		FullTimestamp:          true,
		CallerPrettyfier: func(f *runtime.Frame) (function string, file string) {
			_, line := f.Func.FileLine(f.PC)
			fn := fmt.Sprintf("(%s|%d)", f.Func.Name(), line)
			return fn + " >>>", ""
		},
	})
	Log.SetLevel(logrus.InfoLevel)
	if os.Getenv("ENV") != "prod" {
		Log.SetLevel(logrus.DebugLevel)
	}
}
