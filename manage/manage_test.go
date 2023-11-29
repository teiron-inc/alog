package manage

import (
	"testing"

	"github.com/teiron-inc/alog/log"
)

func TestConsole(t *testing.T) {
	var config log.LogConfig
	config.Console.Item.TimeTmpl = log.DefaultConsoleTimeTmpl
	config.Console.Item.Tmpl = log.DefaultConsoleTmpl
	manage := NewLogManage(&config, nil)
	manage.Console(log.INFO, log.DefaultTag, "Hello,world")
	manage.Consolef(log.DEBUG, log.DefaultTag, "Console output:%s", "Hello,world")
}
