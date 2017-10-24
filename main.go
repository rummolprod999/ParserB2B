package main

import "fmt"

var Addtender = 0

func init() {
	CreateLogFile()
	GetSetting()

}

func main(){
	defer SaveStack()
	Logging("Start parsing")
	Parser()
	Logging("End parsing")
	Logging(fmt.Sprintf("Добавили тендеров %d", Addtender))
}
