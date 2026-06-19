package models

type TNGReport struct {
	Date            string `csv:"F"`
	Status          string `csv:"Status"`
	TransactionType string `csv:"Transaction Type"`
	Reference       string `csv:"Reference"`
	Description     string `csv:"Description"`
	Details         string `csv:"Details"`
	Amount          string `csv:"Amount(RM)"`
}
