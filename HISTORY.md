# HISTORY

## v1.2.0
  by zwj186
* 增加自动重载配置支持
* 修改重置加载配置文件处理细节,完善重载前缺少释放处理
* 修改优化日志按级别保存文件处理性能问题
* 增加外部保存日志接口，以便指定情况保存日志
* 修改读取缓冲区时间间隔单位为毫秒,默认值改为1000毫秒(默认值1秒不变)
* 模板配置项增加Level全大写LEVEL和全小写level配置支持
* 修改优化部分代码细节，删除config.go
* 示例增加外部事件信号退出保存日志处理

## V1.1.2

* 文件存储增加清理机制（清理时间间隔、保留最近日志（以天为单位））

## v1.1.1

* 修改配置文件实现

## v1.1.0

* 增加针对`ElasticSearch`的持久化存储
* 增加针对`MongoDB`的持久化存储
* 优化一些实现及错误处理

## v1.0.6

* 针对`Global`增加`Level`配置
* 针对`manage`中的一些实现方式进行调整

## v1.0.5

* 修复一些关于配置`bug`
* 增加`IsEnabled`配置是否启用日志

## v1.0.4

* 针对`ALog`中的日志级别函数进行稍大强度的调整
* 规范`Global`日志级别函数输出
* 增加`NewALog`函数，运行外部创建`ALog`实例

## v1.0.3

* 增加`ALog`结构体，提供统一的处理函数
* 加入`ShortName`(短文件名)日志项模板格式
* 调整`README.md`

## v1.0.2

* `target`可以指向多个store,以`,`分隔
* 增加针对`FileCaller`的配置
* 在`Tags`配置下，增加`level`配置
* 针对范例程序进行更细致的划分，单独抽出`config`