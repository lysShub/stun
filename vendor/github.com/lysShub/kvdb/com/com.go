package com

import (
	"errors"
	"os"
	"path/filepath"
)

// 方法可执行文件(不包括)所在路径
func GetExePath() string {
	ex, err := os.Executable()
	if err != nil {
		exReal, err := filepath.EvalSymlinks(ex)
		if err != nil {
			dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
			if err != nil {
				return "./"
			}
			return dir
		}
		return filepath.Dir(exReal)
	}
	return filepath.Dir(ex)

}

func ExpressionCalculate(exp string, lv int, rgt []byte) (bool, error) {
	var fv int = 0
	for i := 0; i < len(rgt); i++ {
		fv = fv + int(rgt[i])<<((len(rgt)-i)*8)
	}

	if exp == `>` {
		if lv > fv {
			return true, nil
		}
		return false, nil
	} else if exp == `>=` {
		if lv >= fv {
			return true, nil
		}
		return false, nil
	} else if exp == `<` {
		if lv < fv {
			return true, nil
		}
		return false, nil
	} else if exp == `<=` {
		if lv <= fv {
			return true, nil
		}
		return false, nil
	} else if exp == `!=` {
		if lv != fv {
			return true, nil
		}
		return false, nil
	} else if exp == `=` {
		if lv == fv {
			return true, nil
		}
		return false, nil
	} else {
		return false, errors.New(`invalid expression`)
	}
}
