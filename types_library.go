package main

type AccessToken struct {
	Token string `xml:",chardata"`
}

type ListProcedures struct {
	ListProc []ItemInList `xml:"item"`
}

type ItemInList struct {
	Id   string `xml:"id"`
	Date int64  `xml:"date"`
}

type TradeProc struct {
	Id          string `xml:"id"`
	Number      string `xml:"number"`
	DateBegin   int64  `xml:"date_begin"`
	PublishDate int64  `xml:"publish_date"`
	ChangeDate  int64  `xml:"change_date"`
}
