// Copyright 2017 The go-vgo Project Developers. See the COPYRIGHT
// file at the top-level directory of this distribution and at
// https://github.com/go-vgo/gt/blob/master/LICENSE
//
// Licensed under the Apache License, Version 2.0 <LICENSE-APACHE or
// http://www.apache.org/licenses/LICENSE-2.0> or the MIT license
// <LICENSE-MIT or http://opensource.org/licenses/MIT>, at your
// option. This file may not be copied, modified, or distributed
// except according to those terms.

package zlog

import (
	// "errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Zlog struct {
	// log.Logger
}

var (
	logger, errLogger *zap.Logger
	sugar, errSugar   *zap.SugaredLogger
	zErr              error
	zlogTime          zapcore.Field = zap.String("time", time.Now().Format("2006-01-02 15:04:05"))
)

type logConfig struct {
	Mode    string
	Path    string
	Name    string
	MaxDays int64
	// Srv  Server     `toml:"server"`
}

var config logConfig

func Init(tpath string) {
	if _, err := toml.DecodeFile(tpath, &config); err != nil {
		fmt.Println(err)
		return
	}

	go deleteOldLog()

	if config.Mode == "dev" {
		InitDev()
		zlogTime = zap.Error(nil)
	} else {
		InitLog()
		InitErrLog()
		// go InitErrLog()
	}
}

func deleteOldLog() {
	fileDir, _ := conf()
	var maxDays int64 = 28

	if config.MaxDays != 0 {
		maxDays = config.MaxDays
	}

	filepath.Walk(fileDir, func(path string, info os.FileInfo, err error) (returnErr error) {
		defer func() {
			if r := recover(); r != nil {
				returnErr = fmt.Errorf("Unable to delete old log '%s', error: %+v", path, r)
			}
		}()

		if info.IsDir() && info.ModTime().Unix() < (time.Now().Unix()-60*60*24*maxDays) {

			if strings.HasPrefix(filepath.Base(path), filepath.Base(fileDir)) {
				// if err := os.Remove(path); err != nil {
				if err := os.RemoveAll(path); err != nil {
					returnErr = fmt.Errorf("Failed to remove %s: %v", path, err)
				}
			}
		}
		return returnErr
	})
}

func InitDev() {
	// logger, _ = zap.NewProduction()
	logCfg := zap.NewDevelopmentConfig()
	logCfg.Sampling = nil
	logger, zErr = logCfg.Build()
	if zErr != nil {
		log.Fatal("NewDevelopmentConfig ERR:", zErr)
	}

	errLogger = logger

	defer logger.Sync() // flushes buffer, if any
	sugar = logger.Sugar()
	errSugar = sugar
}

func conf() (string, string) {
	// var lpath, name string
	var lpath, name string = "./log", "log"

	if config.Path != "" {
		lpath = config.Path
	}

	if config.Name != "" {
		name = config.Name
	}

	return lpath, name
}

func InitLog() {
	lpath, name := conf()

	logTime := time.Now().Format("2006-01-02")
	logPath := lpath + "/" + logTime + "/" + name + ".json"
	ws := zapcore.AddSync(&lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    500, // megabytes
		MaxBackups: 3,
		MaxAge:     28, // days
	})
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		ws,
		zap.InfoLevel,
	)
	// logger = zap.New(core).WithOptions(zap.AddCaller())
	logger = zap.New(core).WithOptions(zap.AddStacktrace(zap.InfoLevel))

	defer logger.Sync() // flushes buffer, if any
	sugar = logger.Sugar()
}

func InitErrLog() {
	// lumberjack.Logger is already safe for concurrent use, so we don't need to
	// lock it.
	lpath, name := conf()

	logTime := time.Now().Format("2006-01-02")
	logPath := lpath + "/" + logTime + "/" + name + "_err.json"
	ws := zapcore.AddSync(&lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    500, // megabytes
		MaxBackups: 3,
		MaxAge:     28, // days
	})

	highPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= zapcore.ErrorLevel
	})

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		ws,
		// zap.ErrorLevel,
		highPriority,
	)

	errLogger = zap.New(core).WithOptions(zap.AddStacktrace(zap.ErrorLevel))
	defer logger.Sync() // flushes buffer, if any
	errSugar = errLogger.Sugar()
}

func Print(args ...interface{}) string {
	if len(args) == 0 {
		return ""
	}
	return fmt.Sprintf("%v ", args[0])
}

func Printf(args ...interface{}) string {
	if len(args) < 5 {
		return ""
	}

	return fmt.Sprintf(
		"method: %v, statusCode: %v, req: %s, ip: %s, time: %fs",
		args[0],
		args[1],
		args[2],
		args[3],
		args[4])
}

func (z *Zlog) Error(msg string, err error) {
	errLogger.Error(msg,
		zlogTime,
		zap.Error(err),
	)
}

func LogInfo(msg string, info ...string) {
	var logInfo string = ""
	if len(info) > 0 {
		logInfo = info[0]
	}

	errLogger.Info(msg,
		zlogTime,
		zap.String("info", logInfo),
	)
}

func Error(msg string, err ...error) {
	var logErr error = nil
	if len(err) > 0 {
		logErr = err[0]
	}
	errLogger.Error(msg,
		zlogTime,
		zap.Error(logErr),
	)
}

func Fatal(msg string, err ...error) {
	var logErr error = nil
	if len(err) > 0 {
		logErr = err[0]
	}
	errLogger.Fatal(msg,
		zlogTime,
		zap.Error(logErr),
	)
}

func Panic(msg string, err ...error) {
	var logErr error = nil
	if len(err) > 0 {
		logErr = err[0]
	}
	errLogger.Panic(msg,
		zlogTime,
		zap.Error(logErr),
	)
}

func LogsError(msg string, err error) {
	errSugar.Error(msg,
		zlogTime,
		zap.Error(err),
	)
}

func SugarError(msg string, err error) {
	errSugar.Error(msg,
		zlogTime,
		zap.Error(err),
	)
}

func SugarFatal(msg string, err error) {
	errSugar.Fatal(msg,
		zlogTime,
		zap.Error(err),
	)
}

func SugarPanic(msg string, err error) {
	errSugar.Panic(msg,
		zlogTime,
		zap.Error(err),
	)
}

func Info(msg string, info ...string) {
	var logInfo string = ""
	if len(info) > 0 {
		logInfo = info[0]
	}
	logger.Info(msg,
		zlogTime,
		zap.String("info", logInfo),
	// fields,
	)
}

func Warn(msg string, warn ...string) {
	var logWarn string = ""
	if len(warn) > 0 {
		logWarn = warn[0]
	}
	logger.Warn(msg,
		zlogTime,
		zap.String("warn", logWarn),
	)
}

func Debug(msg string, debug ...string) {
	var logDebug string = ""
	if len(debug) > 0 {
		logDebug = debug[0]
	}
	logger.Debug(msg,
		zlogTime,
		zap.String("debug", logDebug),
	)
}

func Infoff(msg string, fields ...zapcore.Field) {
	logger.Info(msg,
		zlogTime,
		fields[0],
	)
}

func LogError(msg string, err error) {
	logger.Error(msg,
		zlogTime,
		zap.Error(err),
	)
}

func LogPanic(msg string, err error) {
	logger.Panic(msg,
		zlogTime,
		zap.Error(err),
	)
}

func LogFatal(msg string, err error) {
	logger.Fatal(msg,
		zlogTime,
		zap.Error(err),
	)
}

func Infof(msg, info string) {
	sugar.Infof(msg,
		zlogTime,
		zap.String("info", info),
	)
}

func InfoW(msg, info string) {
	sugar.Infow(msg,
		zlogTime,
		"info", info,
	)
}

func Errorf(msg string, err error) {
	sugar.Errorf(msg,
		zlogTime,
		zap.Error(err),
	)
}

func Warnf(msg, warn string) {
	sugar.Warnf(msg,
		zlogTime,
		zap.String("warn", warn),
	)
}