package ryt

type RytReport struct {
	Date        string `csv:"Date"`
	Description string `csv:"Description"`
	Amount      string `csv:"(MYR) Amount"`
}
