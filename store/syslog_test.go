package store

import (
	"testing"
	"time"

	"github.com/teiron-inc/alog/log"
)

func TestSyslogStore(t *testing.T) {
	var cfg log.SyslogConfig
	cfg.Tag = ""
	store := NewSyslogStore(cfg)
	var err error
	for i := 0; i < 10; i++ {
		var item log.LogItem
		item.ID = uint64(i)
		item.Time = time.Now()
		item.Level = log.INFO
		item.Tag = log.DefaultTag
		item.Message = "Write test.........................."
		err = store.Store(&item)
		if err != nil {
			break
		}
	}
	if err != nil {
		t.Error(err)
		return
	}
	t.Log("Write success.")
}
