package comm

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	. "github.com/luyu6056/leaf/log"
)

//获取调用 上一层堆栈
func GetFileLine() string {
	return GetFileLineBySkip(3)
}

func GetFileLineBySkip(skip int) string {
	_, file, line, _ := runtime.Caller(skip)
	return fmt.Sprintf("%v:%v", file, line)
}

func Recover() {
	if err := recover(); err != nil {
		fmt.Println(err)
		stack := debug.Stack()
		Error(string(stack))
	}
}

func Try(f func()) {
	defer Recover()
	f()
}

func String2Int32(istr string) int32 {
	i, _ := strconv.Atoi(istr)
	return int32(i)
}

//判断int32数组内容是否一样
func SliceInt32IsEqual(a, b *[]int32) bool {
	if len(*a) != len(*b) {
		return false
	}
	for _, v := range *a {
		is := false
		for _, v1 := range *b {
			if v == v1 {
				is = true
				break
			}
		}
		if !is {
			return false
		}
	}
	return true
}

func SliceInt32toString(s []int32) string {
	str := ""
	for _, v := range s {
		if str == "" {
			str = strconv.Itoa(int(v))
		} else {
			str = str + "," + strconv.Itoa(int(v))
		}
	}
	return str
}

func SliceInt32Find(s []int32, i int32) bool {
	for _, v := range s {
		if i == v {
			return true
		}
	}
	return false
}

func GoRoutineExecFunction(f func()) {
	go func() {
		defer func() { // 必须要先声明defer，否则不能捕获到panic异常
			if err := recover(); err != nil {
				stack := debug.Stack()
				Error(string(stack))
			}
		}()
		f()
	}()
}

//几个时间相关的函数
func GetLocalDiffTime() int {
	t := time.Unix(0, 0)
	return t.Hour()*3600 + t.Minute()*60 + t.Second()
}

//获取当前时间字符串
func GetTimeString() string {
	return GetTimeStringByUTC(time.Now().Unix())
}

//获取utc时间字符串
func GetTimeStringByUTC(utc int64) string {
	t := time.Unix(utc, 0)
	return fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
}
func GetTimeByString(tstr string) time.Time {
	if tstr <= "1970-01-01 00:00:00" {
		return time.Unix(0, 0)
	}
	//string转化为时间，layout必须为 "2006-01-02 15:04:05"
	t, err := time.Parse("2006-01-02 15:04:05", tstr)
	if err != nil {
		fmt.Printf("GetTimeByString err %v", err)
	}
	return time.Unix(t.Unix()-int64(GetLocalDiffTime()), 0)
}

func GetFlushTime(t64 int64) int64 {
	flushtime := GetBeginTime(t64) + EveryDay_FlushTime*3600
	if time.Unix(t64, 0).Hour() < EveryDay_FlushTime {
		flushtime -= 86400
	}
	return flushtime
}

func GetTodayFlushTime() int64 {
	return GetFlushTime(time.Now().Unix())
}

//获取今日0点时间戳
func GetTodayBeginTime() int64 {
	return GetBeginTime(time.Now().Unix())
}

//获取0点时间戳
func GetBeginTime(t64 int64) int64 {
	t := time.Unix(t64, 0)
	return t.Unix() - int64(t.Hour()*3600+t.Minute()*60+t.Second())
}

//获取本周星期一早上五点时间
func GetWeekFlushTime() int64 {
	begin := GetWeekBeginTime()
	if time.Now().Weekday() == time.Monday {
		if time.Now().Hour() < EveryDay_FlushTime {
			begin -= 86400 * 7
		}
	}
	return begin + EveryDay_FlushTime*3600
}

//获取本周一0点时间戳
func GetWeekBeginTime() int64 {
	begin := GetTodayBeginTime()
	return begin - (int64(GetTimeWeekday1to7())-1)*86400
}

//添加时间未来整分钟的时间,方便定时器处理
func GetAddNowTimeForMinuteBegin(addsecond int32, now ...int64) time.Time {
	if len(now) > 0 {
		t := now[0] + int64(addsecond)
		return time.Unix(t-t%60, 0)
	}
	t := time.Now().Unix() + int64(addsecond)
	return time.Unix(t-t%60, 0)
}

//获取年月,如:201703
func GetYearMonth() int {
	t := time.Now()
	s := fmt.Sprintf("%d%02d", t.Year(), t.Month())
	i, _ := strconv.Atoi(s)
	return i
}

func GetLogFlag() int {
	return log.LstdFlags | log.Lshortfile
}

func GetMD5(str string) string {
	hash := md5.New()
	hash.Write([]byte(str))
	return hex.EncodeToString(hash.Sum(nil))
}

//gameserver和loginserver通信签名用的key
const loginServerSignKey = "abcdefg"

func GetLoginServerSign(server_id int32) string {
	return GetMD5(loginServerSignKey + strconv.Itoa(int(server_id)))
}

func GetTimeWeekday1to7() int32 {
	wd := time.Now().Weekday()
	if wd == time.Sunday {
		return 7
	} else {
		return int32(wd)
	}
}

func DaemonClose() {
	if runtime.GOOS != "linux" {
		return
	}
	for i := 0; i < len(os.Args); i++ {
		param := os.Args[i]
		idx := strings.Index(param, "daemon")
		if idx >= 0 {
			s := strings.Split(param, "daemon")
			if len(s) >= 2 {
				parentpid, _ := strconv.Atoi(s[1])
				cmdstr := fmt.Sprintf("kill -9 %v", parentpid)
				cmd := exec.Command("/bin/sh", "-c", cmdstr)
				err := cmd.Start()
				if err != nil {
					println("err:", err.Error())
				} else {
					println("cmd:", cmdstr, "success!")
				}
			}
		}
	}
}

func GetSvnVerson() int32 {
	filepaths := []string{"./data/version.txt", "./version.txt", "./Lua/version.txt"}
	for i := 0; i < len(filepaths); i++ {
		data, err := ioutil.ReadFile(filepaths[i])
		if err != nil {
			Error("%v", err.Error())
			continue
		}
		filedata := string(data)
		s := strings.Split(filedata, "\n")
		for _, v := range s {
			f := func(sep string) int32 {
				idx := strings.Index(v, sep)
				if idx == 0 {
					istr := strings.TrimSpace(v[len(sep):])
					i32 := String2Int32(istr)
					return i32
				}
				return 0
			}
			i32 := f("版本:")
			if i32 > 0 {
				return i32
			}
			i32 = f("Revision:")
			if i32 > 0 {
				return i32
			}
		}
	}
	Error("GetSvnVerson nofind verson file")
	return 0
}

//创建守护进程
func Daemon(logdir string) {
	os.MkdirAll("log", 0777)

	if runtime.GOOS != "linux" {
		return
	}

	var daemon bool
	for i := 0; i < len(os.Args); i++ {
		if os.Args[i] == "child" {
			return
		}
		if os.Args[i] == "daemon" {
			daemon = true
		}
	}

	if !daemon {
		//启动守护进程
		cmd := exec.Command(os.Args[0], "daemon")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Start()
		if err != nil {
			str := fmt.Sprintf("%s 启动失败 %v\n", GetTimeString(), err)
			println(str)
			return
		}
		os.Exit(0)
	}

	for {
		cmd := exec.Command(os.Args[0], "child", fmt.Sprintf("daemon%v", os.Getpid()))
		//将其他命令传入生成出的进程
		cmd.Stdin = os.Stdin
		//给新进程设置文件描述符，可以重定向到文件中
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		var err error
		var file *os.File
		if logdir != "" {
			file, err = os.OpenFile(fmt.Sprintf("%v_out.log", logdir), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
			if err == nil {
				cmd.Stdout = file
				cmd.Stderr = file
			}
		}
		err = cmd.Start()
		if err != nil {
			str := fmt.Sprintf("%s 启动失败 %v\n", GetTimeString(), err)
			println(str)
			if file != nil {

				file.Write([]byte(str))
				file.Close()
			}
			break
		}
		str := fmt.Sprintf("%s:%s 启动\n", GetTimeString(), os.Args[0])
		println(str)
		if file != nil {
			file.Write([]byte(str))
		}

		err = cmd.Wait()
		str = fmt.Sprintf("%s:%s 退出 err [%v]\n", GetTimeString(), os.Args[0], err)
		println(str)
		if file != nil {
			file.Write([]byte(str))
			file.Close()
		}
		//if err.Error() == "signal: killed" {
		//	os.Exit(0) //如果子进程是被kill掉的,就不监控了,结束吧
		//}
		time.Sleep(time.Second * 5)
	}
}
