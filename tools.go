package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var layout = "2006-01-02T15:04:05"

func SaveStack() {
	if p := recover(); p != nil {
		var buf [4096]byte
		n := runtime.Stack(buf[:], false)
		file, err := os.OpenFile(string(FileLog), os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
		defer file.Close()
		if err != nil {
			fmt.Println("Ошибка записи stack log", err)
			return
		}
		fmt.Fprintln(file, fmt.Sprintf("Fatal Error %v", p))
		fmt.Fprintf(file, "%v  ", string(buf[:n]))
	}

}

func GetOkpd(s string) (int, string) {
	okpd2GroupCode := 0
	okpd2GroupLevel1Code := ""
	if len(s) > 1 {
		if strings.Index(s, ".") != -1 {
			okpd2GroupCode, _ = strconv.Atoi(s[:2])
		} else {
			okpd2GroupCode, _ = strconv.Atoi(s[:2])
		}
	}
	if len(s) > 3 {
		if strings.Index(s, ".") != -1 {
			okpd2GroupLevel1Code = s[3:4]
		}
	}
	return okpd2GroupCode, okpd2GroupLevel1Code
}

func GetToken() string {
	var st string
	url := fmt.Sprintf("https://www.b2b-center.ru/integration/xml/User.Login?login=%s&password=%s", User, Pass)
	s := DownloadPage(url)
	var tkn AccessToken
	if err := xml.Unmarshal([]byte(s), &tkn); err != nil {
		Logging("Ошибка при парсинге строки", err)
		return st
	}
	return tkn.Token
}

func GetListProcedures(token string) string {
	var st string
	tNow := time.Now()
	tEnd := time.Now().Unix()
	tStart := tNow.Add(time.Hour * -165).Unix()
	url := fmt.Sprintf("https://www.b2b-center.ru/integration/xml/TradeProcedures.GetList?access_token=%s&date_from=%v&date_to=%v", token, tStart, tEnd)
	st = DownloadPage(url)
	return st
}

func DownloadPage(url string) string {
	var st string
	count := 0
	for {
		if count > 50 {
			Logging(fmt.Sprintf("Не скачали файл за %d попыток", count))
			return st
		}
		st = GetPage(url)
		if st == "" {
			count++
			Logging("Получили пустую страницу", url)
			continue
		}

		return st
	}
}

func GetPage(url string) string {
	var st string
	resp, err := http.Get(url)
	defer resp.Body.Close()
	if err != nil {
		Logging("Ошибка скачивания", url, err)
		return st
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		Logging("Ошибка чтения", url, err)
		return st
	}

	return string(body)
}
