package manage

import (
	"fmt"
	"go/build"
	"math"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"text/template"
	"time"

	"github.com/teiron-inc/alog/buffer"
	"github.com/teiron-inc/alog/log"
	"github.com/teiron-inc/alog/store"
	"github.com/teiron-inc/alog/utils"
)

// NewLogManage 创建新的LogManage实例
func NewLogManage(config *log.LogConfig, cfgFile *log.ConfigFile) log.LogManage {
	manage := &_LogManage{
		Config:  config,
		running: true,
		cfile:   cfgFile,
	}
	manage.Template = map[log.TmplKey]*template.Template{
		log.TmplConsole:     template.Must(template.New("").Parse(config.Console.Item.Tmpl)),
		log.TmplConsoleTime: template.Must(template.New("").Parse(config.Console.Item.TimeTmpl)),
	}
	switch config.Global.Buffer.Engine {
	case log.REDIS_BUFFER:
		var redisConfig log.RedisConfig
		redisStore := config.Store.Redis
		if redisStore != nil {
			if v, ok := redisStore[config.Global.Buffer.TargetStore]; ok {
				redisConfig = v
			}
		}
		manage.Buffer = buffer.NewRedisBuffer(redisConfig)
	default:
		manage.Buffer = buffer.NewMemoryBuffer()
	}
	manageStore := make(map[string]log.LogStore)
	if fileStore := config.Store.File; fileStore != nil {
		for k, v := range fileStore {
			manageStore[k] = store.NewFileStore(v)
		}
	}
	if elasticStore := config.Store.Elastic; elasticStore != nil {
		for k, v := range elasticStore {
			manageStore[k] = store.NewElasticStore(v)
		}
	}
	if mongoStore := config.Store.Mongo; mongoStore != nil {
		for k, v := range mongoStore {
			manageStore[k] = store.NewMongoStore(v)
		}
	}
	if syslogStore := config.Store.Syslog; syslogStore != nil {
		for k, v := range syslogStore {
			manageStore[k] = store.NewSyslogStore(v)
		}
	}
	manage.Store = manageStore
	manage.logDay = getLogDayTime()
	manage.pathLen = len(build.Default.GOPATH + "/src/")

	if config.Global.IsEnabled == 1 {
		go manage.execStore()
	}

	return manage
}

// _LogManage
type _LogManage struct {
	sync.RWMutex
	gID        uint64
	storeTotal int64
	running    bool
	cfile      *log.ConfigFile
	logDay     int64
	Config     *log.LogConfig
	Template   map[log.TmplKey]*template.Template
	Buffer     log.LogBuffer
	Store      map[string]log.LogStore
	Tafunc     *time.Timer
	pathLen    int
}

func (lm *_LogManage) Write(level log.LogLevel, tag log.LogTag, v ...interface{}) {
	if lm.Config.Global.Level > level {
		return
	}
	msg := fmt.Sprint(v...)
	lm.writeMsg(level, tag, msg)
}

func (lm *_LogManage) Writef(level log.LogLevel, tag log.LogTag, format string, v ...interface{}) {
	if lm.Config.Global.Level > level {
		return
	}
	msg := fmt.Sprintf(format, v...)
	lm.writeMsg(level, tag, msg)
}

func (lm *_LogManage) Console(level log.LogLevel, tag log.LogTag, v ...interface{}) {
	if lm.Config.Console.Level > level {
		return
	}
	msg := fmt.Sprint(v...)
	lm.writeConsole(level, tag, msg)
}

func (lm *_LogManage) Consolef(level log.LogLevel, tag log.LogTag, format string, v ...interface{}) {
	if lm.Config.Console.Level > level {
		return
	}
	msg := fmt.Sprintf(format, v...)
	lm.writeConsole(level, tag, msg)
}

func (lm *_LogManage) writeConsole(level log.LogLevel, tag log.LogTag, msg string) {
	item := lm.logItem(level, tag, msg)
	lm.console(&item)
}

func (lm *_LogManage) TotalNum() int64 {
	return lm.storeTotal
}

func (lm *_LogManage) writeMsg(level log.LogLevel, tag log.LogTag, msg string) {
	item := lm.logItem(level, tag, msg)
	if lm.isPrint(&item) {
		lm.console(&item)
	}
	lm.Buffer.Push(item)
}

func (lm *_LogManage) logItem(level log.LogLevel, tag log.LogTag, msg string) log.LogItem {
	item := log.LogItem{
		ID:      atomic.AddUint64(&lm.gID, 1),
		Level:   level,
		Time:    time.Now(),
		Tag:     tag,
		Message: msg,
	}
	if lm.Config.Global.ShowFile == 1 {
		item.File = lm.file()
	}
	return item
}

func (lm *_LogManage) file() log.LogFile {
	var logFile log.LogFile
	pc, file, line, ok := runtime.Caller(lm.Config.Global.FileCaller)
	if !ok {
		logFile.FullName = "???"
		logFile.RelativeName = "???"
		logFile.FuncName = "???"
		return logFile
	}
	logFile.FullName = file
	logFile.ShortName = utils.SubstrByStartAfter(file, "/")
	if lm.pathLen > 0 {
		logFile.RelativeName = file[lm.pathLen:]
	} else {
		logFile.RelativeName = logFile.ShortName
	}
	logFile.Line = line
	logFile.FuncName = utils.SubstrByStartAfter(runtime.FuncForPC(pc).Name(), "/")
	return logFile
}

// ReloadConfig 重置加载配置文件
func (lm *_LogManage) ReloadConfig(cfg string) error {
	log.ResetDefaultConfig(lm.Config)
	err := utils.NewConfig(cfg).Read(lm.Config)
	if err != nil {
		return err
	}
	return nil
}

func (lm *_LogManage) isPrint(item *log.LogItem) bool {
	var isPrint bool
	switch lm.Config.Global.Rule {
	case log.AlwaysRule:
		isPrint = lm.isEitherTrue(item, func(lm *_LogManage, item *log.LogItem) bool {
			return lm.Config.Global.IsPrint == 1
		}, func(lm *_LogManage, item *log.LogItem) bool {
			var v bool
			lm.itemTags(item, func(t log.TagConfig) bool {
				if t.Config.IsPrint == 1 {
					v = true
					return true
				}
				return false
			})
			return v
		}, func(lm *_LogManage, item *log.LogItem) bool {
			var v bool
			lm.itemLevels(item, func(l log.LevelConfig) bool {
				if l.Config.IsPrint == 1 {
					v = true
					return true
				}
				return false
			})
			return v
		})
	case log.GlobalRule:
		isPrint = lm.Config.Global.IsPrint == 1
	case log.TagRule:
		lm.itemTags(item, func(t log.TagConfig) bool {
			if t.Config.IsPrint == 1 {
				isPrint = true
				return true
			}
			return false
		})
	case log.LevelRule:
		lm.itemLevels(item, func(l log.LevelConfig) bool {
			if l.Config.IsPrint == 1 {
				isPrint = true
				return true
			}
			return false
		})
	}
	return isPrint
}

func (lm *_LogManage) isEitherTrue(item *log.LogItem, fns ...func(*_LogManage, *log.LogItem) bool) bool {
	for _, fn := range fns {
		if fn(lm, item) {
			return true
		}
	}
	return false
}

func (lm *_LogManage) itemTags(item *log.LogItem, fn func(log.TagConfig) bool) {
	for _, tag := range lm.Config.Tags {
		for _, tagName := range tag.Names {
			if tagName == (*item).Tag {
				if fn(tag) {
					return
				}
				break
			}
		}
	}
}

func (lm *_LogManage) itemLevels(item *log.LogItem, fn func(log.LevelConfig) bool) {
	for _, lev := range lm.Config.Levels {
		for _, levVal := range lev.Values {
			if (*item).Level == levVal {
				if fn(lev) {
					return
				}
				break
			}
		}
	}
}

func (lm *_LogManage) console(item *log.LogItem) {
	lm.stdout(lm.Template[log.TmplConsole], lm.Template[log.TmplConsoleTime], item)
}

func (lm *_LogManage) systemLog(tag log.LogLevel, msg string) {
	item := lm.logItem(tag, "SYSTEM", msg)
	lm.stdout(log.DefaultSystemTmpl, log.DefaultConsoleTimeTmpl, &item)
}

func (lm *_LogManage) stdout(tmpl, timetmpl interface{}, item *log.LogItem) {
	info := log.ParseLogItem(tmpl, timetmpl, item)
	os.Stdout.WriteString(info)
}

func (lm *_LogManage) execStore() {
	interval := time.Duration(lm.Config.Global.Interval) * time.Millisecond
	time.AfterFunc(interval, func() {
		logDay := getLogDayTime()
		if lm.logDay != logDay {
			for _, store := range lm.Store {
				lm.closeStore(store)
			}
			lm.logDay = logDay
		}
		lm.store()
		if lm.running {
			if lm.Config.Global.IsReload {
				fileInfo, serr := os.Stat(lm.cfile.FilePath)
				if serr == nil {
					ModTime := fileInfo.ModTime().Unix()
					if lm.cfile.ModTime != ModTime {
						lm.ReloadConfig(lm.cfile.FilePath)
						lm.cfile.ModTime = ModTime
					}
				}
			}
			lm.execStore()
		} else {
			for _, store := range lm.Store {
				lm.closeStore(store)
			}
		}
	})
}

func (lm *_LogManage) Stored() {
	lm.store()
}

func (lm *_LogManage) Stoped() {
	lm.running = false
}

func (lm *_LogManage) store() {
	for {
		item, err := lm.Buffer.Pop()
		if err != nil {
			panic(err)
		}
		if item == nil {
			break
		}
		targets := lm.storeTargets(item)
		l := len(targets)
		if l == 0 {
			continue
		}
		for i, l := 0, len(targets); i < l; i++ {
			lm.writeStore(targets[i], item)
		}
		atomic.AddInt64(&lm.storeTotal, 1)
		if item.Level == log.FATAL {
			os.Exit(1)
		}
	}
}

func (lm *_LogManage) target(target map[string]string, ts string) {
	tsa := strings.Split(ts, ",")
	for i := 0; i < len(tsa); i++ {
		t := tsa[i]
		if t == "" {
			continue
		}
		if _, ok := target[t]; !ok {
			target[t] = t
		}
	}
}

func (lm *_LogManage) storeTargets(item *log.LogItem) (targets []string) {
	target := make(map[string]string)
	rule := lm.Config.Global.Rule

	if rule == log.AlwaysRule || rule == log.GlobalRule {
		if lm.Config.Global.Level <= item.Level {
			lm.target(target, lm.Config.Global.TargetStore)
		}
	}
	if rule == log.AlwaysRule || rule == log.TagRule {
		lm.itemTags(item, func(tag log.TagConfig) bool {
			if tag.Level <= item.Level {
				lm.target(target, tag.Config.TargetStore)
			}
			return false
		})
	}
	if rule == log.AlwaysRule || rule == log.LevelRule {
		lm.itemLevels(item, func(lev log.LevelConfig) bool {
			lm.target(target, lev.Config.TargetStore)
			return false
		})
	}

	for k := range target {
		targets = append(targets, k)
	}

	return
}

func (lm *_LogManage) writeStore(target string, item *log.LogItem) {
	store, ok := lm.Store[target]
	if !ok {
		return
	}
	err := store.Store(item)
	if err != nil {
		lm.systemLog(log.FATAL, fmt.Sprintf("Write store error:%s", err.Error()))
	}
}

func (lm *_LogManage) closeStore(store log.LogStore) {
	err := store.Close()
	if err != nil {
		lm.systemLog(log.FATAL, fmt.Sprintf("Write store error:%s", err.Error()))
	}
}

func getLogDayTime() int64 {
	nowTime := time.Now().Unix()
	roundTime := nowTime - int64(math.Mod(float64(nowTime+28800), 86400))
	return roundTime
}
