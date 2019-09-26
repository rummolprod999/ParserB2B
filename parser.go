package main

import (
	"database/sql"
	"encoding/xml"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

func Parser() {
	token = GetToken()
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
	var Dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8&parseTime=true&readTimeout=60m&maxAllowedPacket=0&timeout=60m&writeTimeout=60m&autocommit=true&loc=Local", UserDb, PassDb, Server, Port, DbName)
	db, err := sql.Open("mysql", Dsn)
	defer db.Close()
	db.SetConnMaxLifetime(time.Second * 3600)
	if err != nil {
		Logging("Ошибка подключения к БД", err)
	}
	if len(l.ListProc) == 0 {
		Logging("Нет процедур в файле")
	}
	procedures := make(chan Proc, 3)
	go func(token string) {
		for _, p := range l.ListProc {
			procedures <- Proc{GetProcedure(token, p.Id), p.Date, p.Id}
		}
		close(procedures)

	}(token)
	for x := range procedures {
		e := ParserProc(x.Date, x.Id, db, x.StXml)
		if e != nil {
			Logging("Ошибка парсера в процедуре", e)
			continue
		}

	}
}

func ParserProc(date int64, id string, db *sql.DB, st string) error {
	defer SaveStack()
	PublicationDate := time.Unix(date, 0)
	e := ParserProcedure(PublicationDate, id, db, st)
	if e != nil {
		Logging("Ошибка парсера в тендере", e)
		return e
	}
	return nil
}

func ParserProcedure(date time.Time, id string, db *sql.DB, st string) error {
	defer SaveStack()
	s := st
	if s == "" {
		Logging("Получили пустую строку с процедурой")
		return nil
	}
	var p TradeProc
	if err := xml.Unmarshal([]byte(s), &p); err != nil {
		Logging("Ошибка при парсинге строки", err)
		return err
	}
	PublicationDate := time.Unix(p.PublishDate, 0)
	var DateUpdated = date
	if p.ChangeDate != 0 {
		DateUpdated = time.Unix(p.ChangeDate, 0)
	}
	IdXml := p.Id
	TradeId := p.Number
	if p.OsNumber != "" {
		TradeId = p.OsNumber
	}
	if TradeId == "0" && IdXml != "" {
		p := strings.Index(IdXml, "_")
		if p != -1 {
			TradeId = IdXml[:p]
		}
	}
	if TradeId == "" {
		Logging("TradeId is empty ", fmt.Sprintf("%+v", p))
		return nil
	}
	//DateBegin := time.Unix(p.DateBegin, 0)
	//DateTradeEnd := time.Unix(p.DateTradeEnd, 0)
	/*fmt.Println(date)
	fmt.Println(PublicationDate)
	fmt.Println(p.PublishDate)
	fmt.Println(DateUpdated)
	fmt.Println(p.ChangeDate)
	fmt.Println(DateTradeEnd)
	fmt.Println(p.DateTradeEnd)
	fmt.Println(TradeId)
	fmt.Println("")*/
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
	var cancelStatus = 0
	var updated = false
	if TradeId != "" {
		stmt, err := db.Prepare(fmt.Sprintf("SELECT id_tender, date_version FROM %stender WHERE purchase_number = ? AND cancel=0", Prefix))
		rows, err := stmt.Query(TradeId)
		stmt.Close()
		if err != nil {
			Logging("Ошибка выполения запроса", err)
			return err
		}
		for rows.Next() {
			updated = true
			var idTender int
			var dateVersion time.Time
			err = rows.Scan(&idTender, &dateVersion)
			if err != nil {
				Logging("Ошибка чтения результата запроса", err)
				return err
			}
			//fmt.Println(DateUpdated.Sub(dateVersion))
			if dateVersion.Sub(DateUpdated) <= 0 {
				stmtupd, _ := db.Prepare(fmt.Sprintf("UPDATE %stender SET cancel=1 WHERE id_tender = ?", Prefix))
				_, err = stmtupd.Exec(idTender)
				stmtupd.Close()

			} else {
				cancelStatus = 1
			}

		}
		rows.Close()
		//fmt.Println(cancelStatus)
	}
	Href := p.Url
	//fmt.Println(Href)
	PurchaseObjectInfo := p.Description
	comment := p.Comment
	NoticeVersion := ""
	if len(comment) < 2000 {
		NoticeVersion = comment
	}
	PrintForm := Href
	IdOrganizer := 0
	if p.OrganizerINN != "" {
		stmt, _ := db.Prepare(fmt.Sprintf("SELECT id_organizer FROM %sorganizer WHERE inn = ? AND kpp = ?", Prefix))
		rows, err := stmt.Query(p.OrganizerINN, p.OrganizerKPP)
		stmt.Close()
		if err != nil {
			Logging("Ошибка выполения запроса", err)
			return err
		}
		if rows.Next() {
			err = rows.Scan(&IdOrganizer)
			if err != nil {
				Logging("Ошибка чтения результата запроса", err)
				return err
			}
			rows.Close()
		} else {
			rows.Close()
			stmt, _ := db.Prepare(fmt.Sprintf("INSERT INTO %sorganizer SET full_name = ?, inn = ?, kpp = ?", Prefix))
			res, err := stmt.Exec(p.OrganizerName, p.OrganizerINN, p.OrganizerKPP)
			stmt.Close()
			if err != nil {
				Logging("Ошибка вставки организатора", err)
				return err
			}
			id, err := res.LastInsertId()
			IdOrganizer = int(id)
		}
	}

	IdPlacingWay := 0
	if p.TradeType != "" {
		stmt, _ := db.Prepare(fmt.Sprintf("SELECT id_placing_way FROM %splacing_way WHERE name = ? AND code = ? LIMIT 1", Prefix))
		rows, err := stmt.Query(p.TradeType, p.Type)
		stmt.Close()
		if err != nil {
			Logging("Ошибка выполения запроса", err)
			return err
		}
		if rows.Next() {
			err = rows.Scan(&IdPlacingWay)
			if err != nil {
				Logging("Ошибка чтения результата запроса", err)
				return err
			}
			rows.Close()
		} else {
			rows.Close()
			conf := GetConformity(p.TradeType)
			stmt, _ := db.Prepare(fmt.Sprintf("INSERT INTO %splacing_way SET name = ?, conformity = ?, code = ?", Prefix))
			res, err := stmt.Exec(p.TradeType, conf, p.Type)
			stmt.Close()
			if err != nil {
				Logging("Ошибка вставки placing way", err)
				return err
			}
			id, err := res.LastInsertId()
			IdPlacingWay = int(id)

		}
	}
	IdEtp := 0
	etpName := "Система электронных торгов B2B-Center"
	etpUrl := "http://b2b-center.ru"
	if true {
		stmt, _ := db.Prepare(fmt.Sprintf("SELECT id_etp FROM %setp WHERE name = ? AND url = ? LIMIT 1", Prefix))
		rows, err := stmt.Query(etpName, etpUrl)
		stmt.Close()
		if err != nil {
			Logging("Ошибка выполения запроса", err)
			return err
		}
		if rows.Next() {
			err = rows.Scan(&IdEtp)
			if err != nil {
				Logging("Ошибка чтения результата запроса", err)
				return err
			}
			rows.Close()
		} else {
			rows.Close()
			stmt, _ := db.Prepare(fmt.Sprintf("INSERT INTO %setp SET name = ?, url = ?, conf=0", Prefix))
			res, err := stmt.Exec(etpName, etpUrl)
			stmt.Close()
			if err != nil {
				Logging("Ошибка вставки etp", err)
				return err
			}
			id, err := res.LastInsertId()
			IdEtp = int(id)
		}
	}
	var EndDate = time.Time{}
	if p.DateEnd != 0 {
		EndDate = time.Unix(p.DateEnd, 0)
	}
	typeFz := 3
	idTender := 0
	Version := 0
	UrlXml := Href
	stmtt, _ := db.Prepare(fmt.Sprintf("INSERT INTO %stender SET id_region = 0, id_xml = ?, purchase_number = ?, doc_publish_date = ?, href = ?, purchase_object_info = ?, type_fz = ?, id_organizer = ?, id_placing_way = ?, id_etp = ?, end_date = ?, cancel = ?, date_version = ?, num_version = ?, notice_version = ?, xml = ?, print_form = ?", Prefix))
	rest, err := stmtt.Exec(IdXml, TradeId, PublicationDate, Href, PurchaseObjectInfo, typeFz, IdOrganizer, IdPlacingWay, IdEtp, EndDate, cancelStatus, DateUpdated, Version, NoticeVersion, UrlXml, PrintForm)
	stmtt.Close()
	if err != nil {
		Logging("Ошибка вставки tender", err)
		return err
	}
	idt, err := rest.LastInsertId()
	idTender = int(idt)
	if updated {
		Updatetender++
	} else {
		Addtender++
	}
	var LotNumber = 1
	for _, lot := range p.Lots {
		idLot := 0
		MaxPrice := lot.MaxPrice
		stmt, _ := db.Prepare(fmt.Sprintf("INSERT INTO %slot SET id_tender = ?, lot_number = ?, max_price = ?, currency = ?", Prefix))
		res, err := stmt.Exec(idTender, LotNumber, MaxPrice, p.Currency)
		stmt.Close()
		if err != nil {
			Logging("Ошибка вставки lot", err)
			return err
		}
		id, _ := res.LastInsertId()
		idLot = int(id)
		idCustomer := 0
		if p.OrganizerINN != "" {
			stmt, _ := db.Prepare(fmt.Sprintf("SELECT id_customer FROM %scustomer WHERE inn = ? LIMIT 1", Prefix))
			rows, err := stmt.Query(p.OrganizerINN)
			stmt.Close()
			if err != nil {
				Logging("Ошибка выполения запроса", err)
				return err
			}
			if rows.Next() {
				err = rows.Scan(&idCustomer)
				if err != nil {
					Logging("Ошибка чтения результата запроса", err)
					return err
				}
				rows.Close()
			} else {
				rows.Close()
				out, err := exec.Command("uuidgen").Output()
				if err != nil {
					Logging("Ошибка генерации UUID", err)
					return err
				}
				stmt, _ := db.Prepare(fmt.Sprintf("INSERT INTO %scustomer SET full_name = ?, is223=1, reg_num = ?, inn = ?", Prefix))
				res, err := stmt.Exec(p.OrganizerName, out, p.OrganizerINN)
				stmt.Close()
				if err != nil {
					Logging("Ошибка вставки организатора", err)
					return err
				}
				id, err := res.LastInsertId()
				idCustomer = int(id)
			}
		}
		for _, po := range lot.OkpdItems {
			stmtr, _ := db.Prepare(fmt.Sprintf("INSERT INTO %spurchase_object SET id_lot = ?, id_customer = ?, okpd2_code = ?, name = ?, quantity_value = ?, customer_quantity_value = ?, okei = ?", Prefix))
			_, errr := stmtr.Exec(idLot, idCustomer, po.Item, lot.Name, lot.Quantity, lot.Quantity, lot.UnitName)
			stmtr.Close()
			if errr != nil {
				Logging("Ошибка вставки purchase_object", errr)
				return errr
			}
		}
		for _, cr := range lot.DeliveryPlaces {
			stmt, _ := db.Prepare(fmt.Sprintf("INSERT INTO %scustomer_requirement SET id_lot = ?, id_customer = ?, delivery_place = ?, delivery_term = ?", Prefix))
			_, err := stmt.Exec(idLot, idCustomer, cr.Item, p.PaymentTerms)
			stmt.Close()
			if err != nil {
				Logging("Ошибка вставки customer_requirement", err)
				return err

			}

		}
	}
	e := TenderKwords(db, idTender, &(p.Comment))
	if e != nil {
		Logging("Ошибка обработки TenderKwords", e)
	}

	e1 := AddVerNumber(db, TradeId)
	if e1 != nil {
		Logging("Ошибка обработки AddVerNumber", e1)
	}
	return nil
}

func AddVerNumber(db *sql.DB, RegistryNumber string) error {
	verNum := 1
	mapTenders := make(map[int]int)
	stmt, _ := db.Prepare(fmt.Sprintf("SELECT id_tender FROM %stender WHERE purchase_number = ? ORDER BY UNIX_TIMESTAMP(date_version) ASC", Prefix))
	rows, err := stmt.Query(RegistryNumber)
	stmt.Close()
	if err != nil {
		Logging("Ошибка выполения запроса", err)
		return err
	}
	for rows.Next() {
		var rNum int
		err = rows.Scan(&rNum)
		if err != nil {
			Logging("Ошибка чтения результата запроса", err)
			return err
		}
		mapTenders[verNum] = rNum
		verNum++
	}
	rows.Close()
	for vn, idt := range mapTenders {
		stmtr, _ := db.Prepare(fmt.Sprintf("UPDATE %stender SET num_version = ? WHERE id_tender = ?", Prefix))
		_, errr := stmtr.Exec(vn, idt)
		stmtr.Close()
		if errr != nil {
			Logging("Ошибка вставки NumVersion", errr)
			return err
		}
	}

	return nil
}

func TenderKwords(db *sql.DB, idTender int, comment *string) error {
	resString := ""
	stmt, _ := db.Prepare(fmt.Sprintf("SELECT DISTINCT po.name, po.okpd_name FROM %spurchase_object AS po LEFT JOIN %slot AS l ON l.id_lot = po.id_lot WHERE l.id_tender = ?", Prefix, Prefix))
	rows, err := stmt.Query(idTender)
	stmt.Close()
	if err != nil {
		Logging("Ошибка выполения запроса", err)
		return err
	}
	for rows.Next() {
		var name sql.NullString
		var okpdName sql.NullString
		err = rows.Scan(&name, &okpdName)
		if err != nil {
			Logging("Ошибка чтения результата запроса", err)
			return err
		}
		if name.Valid {
			resString = fmt.Sprintf("%s %s ", resString, name.String)
		}
		if okpdName.Valid {
			resString = fmt.Sprintf("%s %s ", resString, okpdName.String)
		}
	}
	rows.Close()
	stmt1, _ := db.Prepare(fmt.Sprintf("SELECT DISTINCT file_name FROM %sattachment WHERE id_tender = ?", Prefix))
	rows1, err := stmt1.Query(idTender)
	stmt1.Close()
	if err != nil {
		Logging("Ошибка выполения запроса", err)
		return err
	}
	for rows1.Next() {
		var attName sql.NullString
		err = rows1.Scan(&attName)
		if err != nil {
			Logging("Ошибка чтения результата запроса", err)
			return err
		}
		if attName.Valid {
			resString = fmt.Sprintf("%s %s ", resString, attName.String)
		}
	}
	rows1.Close()
	idOrg := 0
	stmt2, _ := db.Prepare(fmt.Sprintf("SELECT purchase_object_info, id_organizer FROM %stender WHERE id_tender = ?", Prefix))
	rows2, err := stmt2.Query(idTender)
	stmt2.Close()
	if err != nil {
		Logging("Ошибка выполения запроса", err)
		return err
	}
	for rows2.Next() {
		var idOrgNull sql.NullInt64
		var purOb sql.NullString
		err = rows2.Scan(&purOb, &idOrgNull)
		if err != nil {
			Logging("Ошибка чтения результата запроса", err)
			return err
		}
		if idOrgNull.Valid {
			idOrg = int(idOrgNull.Int64)
		}
		if purOb.Valid {
			resString = fmt.Sprintf("%s %s ", resString, purOb.String)
		}

	}
	rows2.Close()
	if idOrg != 0 {
		stmt3, _ := db.Prepare(fmt.Sprintf("SELECT full_name, inn FROM %sorganizer WHERE id_organizer = ?", Prefix))
		rows3, err := stmt3.Query(idOrg)
		stmt3.Close()
		if err != nil {
			Logging("Ошибка выполения запроса", err)
			return err
		}
		for rows3.Next() {
			var innOrg sql.NullString
			var nameOrg sql.NullString
			err = rows3.Scan(&nameOrg, &innOrg)
			if err != nil {
				Logging("Ошибка чтения результата запроса", err)
				return err
			}
			if innOrg.Valid {

				resString = fmt.Sprintf("%s %s ", resString, innOrg.String)
			}
			if nameOrg.Valid {
				resString = fmt.Sprintf("%s %s ", resString, nameOrg.String)
			}

		}
		rows3.Close()
	}
	stmt4, _ := db.Prepare(fmt.Sprintf("SELECT DISTINCT cus.inn, cus.full_name FROM %scustomer AS cus LEFT JOIN %spurchase_object AS po ON cus.id_customer = po.id_customer LEFT JOIN %slot AS l ON l.id_lot = po.id_lot WHERE l.id_tender = ?", Prefix, Prefix, Prefix))
	rows4, err := stmt4.Query(idTender)
	stmt4.Close()
	if err != nil {
		Logging("Ошибка выполения запроса", err)
		return err
	}
	for rows4.Next() {
		var innC sql.NullString
		var fullNameC sql.NullString
		err = rows4.Scan(&innC, &fullNameC)
		if err != nil {
			Logging("Ошибка чтения результата запроса", err)
			return err
		}
		if innC.Valid {

			resString = fmt.Sprintf("%s %s ", resString, innC.String)
		}
		if fullNameC.Valid {
			resString = fmt.Sprintf("%s %s ", resString, fullNameC.String)
		}
	}
	rows4.Close()
	resString = fmt.Sprintf("%s %s", resString, *comment)
	re := regexp.MustCompile(`\s+`)
	resString = re.ReplaceAllString(resString, " ")
	stmtr, _ := db.Prepare(fmt.Sprintf("UPDATE %stender SET tender_kwords = ? WHERE id_tender = ?", Prefix))
	_, errr := stmtr.Exec(resString, idTender)
	stmtr.Close()
	if errr != nil {
		Logging("Ошибка вставки TenderKwords", errr)
		return err
	}
	return nil
}
