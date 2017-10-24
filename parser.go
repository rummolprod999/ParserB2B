package main


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
}
