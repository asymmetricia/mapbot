package log

import "github.com/op/go-logging"

var Log = logging.MustGetLogger("mapbot")
var format = logging.MustStringFormatter(
	`%{color}%{time:15:04:05.000} %{shortfunc} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}`,
)

func init() {
	logging.SetFormatter(format)
}
