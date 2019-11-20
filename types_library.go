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
	Id           string     `xml:"id"`
	Number       string     `xml:"number"`
	DateBegin    int64      `xml:"date_begin"`
	DateEnd      int64      `xml:"date_end"`
	PublishDate  int64      `xml:"publish_date"`
	ChangeDate   int64      `xml:"change_date"`
	Url          string     `xml:"url"`
	OsNumber     string     `xml:"os_number"`
	Description  string     `xml:"description"`
	TradeType    string     `xml:"trade_type"`
	Type         string     `xml:"type"`
	DateTradeEnd int64      `xml:"date_trade_end"`
	Comment      string     `xml:"comment"`
	Currency     string     `xml:"currency"`
	PaymentTerms string     `xml:"payment_terms"`
	Lots         []Lot      `xml:"lots>item"`
	Positions    []Position `xml:"positions>item"`
	Organizer
}

type Organizer struct {
	OrganizerName string `xml:"customer>name"`
	OrganizerINN  string `xml:"customer>inn"`
	OrganizerKPP  string `xml:"customer>kpp"`
}

type Lot struct {
	LotId          string          `xml:"id"`
	MaxPrice       string          `xml:"price"`
	Name           string          `xml:"name"`
	Description    string          `xml:"description"`
	Quantity       string          `xml:"quantity"`
	UnitName       string          `xml:"unit_name"`
	OkpdItems      []Okpd          `xml:"okpd>item"`
	Okpd2Items     []Okpd          `xml:"okpd2>item"`
	Okpd3Items     []Okpd          `xml:"okdp>item"`
	DeliveryPlaces []DeliveryPlace `xml:"delivery_place>item"`
}

type Position struct {
	LotId    string `xml:"lot_id"`
	Name     string `xml:"name"`
	Quantity string `xml:"quantity"`
	Price    string `xml:"price_unit"`
	Sum      string `xml:"price_all"`
	UnitName string `xml:"unit_name"`
}

type Okpd struct {
	Item string `xml:",chardata"`
}

type DeliveryPlace struct {
	Item string `xml:",chardata"`
}

type Proc struct {
	StXml string
	Date  int64
	Id    string
}
