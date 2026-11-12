package loggers

import (
	"strings"
)

func getLoggerPackages() map[string]bool {
	return map[string]bool{
		"log": true, "slog": true, "logrus": true, "zap": true,
		"zerolog": true, "logr": true, "glog": true, "klog": true,
	}
}

func isLoggerPackage(to string) bool { return getLoggerPackages()[to[strings.LastIndex(to, "/")+1:]] }
