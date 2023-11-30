package store

import (
	sysLog "log"
	"log/syslog"
	"os"
	"path/filepath"
	"text/template"

	"github.com/teiron-inc/alog/log"
)

// NewSyslogStore 创建基于MongoDB存储的实例
func NewSyslogStore(cfg log.SyslogConfig) log.LogStore {
	var (
		msgTmpl  = cfg.Tmpl
		timeTmpl = cfg.TimeTmpl
		tag      = cfg.Tag
	)
	if msgTmpl == "" {
		msgTmpl = log.DefaultMsgTmpl
	}
	if timeTmpl == "" {
		timeTmpl = log.DefaultTimeTmpl
	}
	if tag == "" {
		tag = filepath.Base(os.Args[0])
	}

	writer, err := syslog.New(syslog.LOG_INFO, tag)
	if err != nil {
		panic(err)
	}
	logger := sysLog.New(writer, "", 0)

	return &SyslogStore{
		logWriter: writer,
		timeTmpl:  template.Must(template.New("").Parse(cfg.TimeTmpl)),
		msgTmpl:   template.Must(template.New("").Parse(cfg.Tmpl)),
		logger:    logger,
	}
}

type SyslogStore struct {
	logWriter *syslog.Writer
	timeTmpl  *template.Template
	msgTmpl   *template.Template
	logger    *sysLog.Logger
}

func (ss *SyslogStore) Store(item *log.LogItem) error {
	logInfo := log.ParseLogItem(ss.msgTmpl, ss.timeTmpl, item)
	ss.logger.Printf(logInfo)
	return nil
}

func (ss *SyslogStore) Close() (err error) {
	return ss.logWriter.Close()
}
