package main

import (
	"database/sql"
	"encoding/xml"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"time"
)

func Parser() {
	token := GetToken()
	if token == "" {
		Logging("Получен пустой токен")
		return
	}
	//fmt.Println(token)
	proc := GetListProcedures(token)
	//fmt.Println(proc)
	if proc == "" || len(proc) < 130 {
		Logging("Получили пустой список торговых процедур")
		return
	}
	var l ListProcedures
	if err := xml.Unmarshal([]byte(proc), &l); err != nil {
		Logging("Ошибка при парсинге строки", err)
		return
	}
	var Dsn = fmt.Sprintf("%s:%s@/%s?charset=utf8&parseTime=true&readTimeout=60m&maxAllowedPacket=0&timeout=60m&writeTimeout=60m&autocommit=true", UserDb, PassDb, DbName)
	db, err := sql.Open("mysql", Dsn)
	defer db.Close()
	db.SetConnMaxLifetime(time.Second * 3600)
	if err != nil {
		Logging("Ошибка подключения к БД", err)
	}
	if len(l.ListProc) == 0 {
		Logging("Нет процедур в файле")
	}
	for _, p := range l.ListProc {
		e := ParserProc(p.Date, p.Id, db, token)
		if e != nil {
			Logging("Ошибка парсера в процедуре", e)
			continue
		}

	}
}

func ParserProc(date int64, id string, db *sql.DB, token string) error {
	defer func() {
		if p := recover(); p != nil {
			Logging(p)
		}
	}()
	PublicationDate := time.Unix(date, 0)
	e := ParserProcedure(PublicationDate, id, db, token)
	if e != nil {
		Logging("Ошибка парсера в тендере", e)
		return e
	}
	return nil
}

func ParserProcedure(PublicationDate time.Time, id string, db *sql.DB, token string) error {
	defer func() {
		if p := recover(); p != nil {
			Logging(p)
		}
	}()
	s := GetProcedure(token, id)
	if s == "" {
		Logging("Получили пустую строку с процедурой")
		return nil
	}
	var p TradeProc
	if err := xml.Unmarshal([]byte(s), &p); err != nil {
		Logging("Ошибка при парсинге строки", err)
		return err
	}
	DateUpdated := PublicationDate
	IdXml := p.Id
	TradeId := p.Number
	DateBegin := time.Unix(p.DateBegin, 0)
	fmt.Println(DateBegin)
	fmt.Println(DateUpdated)
	fmt.Println(TradeId)
	fmt.Println("")
	//fmt.Println(s)
	stmt, _ := db.Prepare(fmt.Sprintf("SELECT id_tender FROM %stender WHERE purchase_number = ? AND date_version = ? AND id_xml = ?", Prefix))
	res, err := stmt.Query(TradeId, DateUpdated, IdXml)
	stmt.Close()
	if err != nil {
		Logging("Ошибка выполения запроса", err)
		return err
	}
	if res.Next() {
		//Logging("Такой тендер уже есть", TradeId)
		res.Close()
		return nil
	}
	res.Close()
	return nil
}
