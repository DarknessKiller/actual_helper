package tng

type TNGReport struct {
	Date            string `csv:"F"`
	Status          string `csv:"Status"`
	TransactionType string `csv:"Transaction Type"`
	Description     string `csv:"Description"`
	Amount          string `csv:"Amount(RM)"`
}
