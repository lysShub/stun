package e

import (
	"io"
	"log"
	"os"
	"runtime"
	"strconv"
)

func t() func(w ...io.Writer) *log.Logger {
	var l *log.Logger // 闭包
	return func(w ...io.Writer) *log.Logger {
		if l == nil || len(w) != 0 {
			if len(w) == 0 && l == nil { // 默认设置为Stderr
				w = []io.Writer{
					os.Stderr,
				}
			}
			l = log.New(io.MultiWriter(w...), "", log.Ldate|log.Ltime)
		}
		return l
	}
}

// L initialize log output.
//
// The impact will be global, wherever you set it.
//
// e.g: e.L(lh, os.Stderr)
var L = t()

// Errlog returns true if there is error(s)
//
// e.g: if e.Errlog(err1,err2){}
func Errlog(err ...error) bool {

	var haveErr bool = false
	for i, e := range err {
		if e != nil {
			haveErr = true
			_, fp, ln, _ := runtime.Caller(1)

			(L()).Println(fp + ":" + strconv.Itoa(ln) + "." + strconv.Itoa(i+1) + "==> " + e.Error())
		}
	}

	return haveErr
}

// ErrExit exit if there is fatal error(s) with code 1998
// defer fun will't be executed
func ErrExit(err ...error) {
	if Errlog(err...) {
		os.Exit(1998)
	}
}
