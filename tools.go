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

func getTimeMoscow(st time.Time) time.Time {
	location, _ := time.LoadLocation("Europe/Moscow")
	p := st.In(location)
	fmt.Println(p)
	return p
}

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
	tStart := tNow.Add(time.Hour * time.Duration(-Count)).Unix()
	url := fmt.Sprintf("https://www.b2b-center.ru/integration/xml/TradeProcedures.GetList?access_token=%s&date_from=%v&date_to=%v", token, tStart, tEnd)
	st = DownloadPage(url)
	return st
}

func DownloadPage(url string) string {
	var st string
	count := 0
	for {
		if count > 10 {
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
	if err != nil {
		Logging("Ошибка response", url, err)
		return st
	}
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

func GetProcedure(token string, idProc string) string {
	var st string
	url := fmt.Sprintf("https://www.b2b-center.ru/integration/xml/TradeProcedures.GetShortTrade?access_token=%s&id=%s", token, idProc)
	st = DownloadPage(url)
	return st

}

func GetConformity(conf string) int {
	s := strings.ToLower(conf)
	switch {
	case strings.Index(s, "открыт") != -1:
		return 5
	case strings.Index(s, "аукцион") != -1:
		return 1
	case strings.Index(s, "котиров") != -1:
		return 2
	case strings.Index(s, "предложен") != -1:
		return 3
	case strings.Index(s, "единств") != -1:
		return 4
	default:
		return 6
	}

}
