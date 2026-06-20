package models

type ActualBudgetReport struct {
	Account       string `csv:"Account"`
	Date          string `csv:"Date"`
	Payee         string `csv:"Payee"`
	Notes         string `csv:"Notes"`
	CategoryGroup string `csv:"Category_Group"`
	Category      string `csv:"Category"`
	Amount        string `csv:"Amount"`
	SplitAmount   string `csv:"Split_Amount"`
	Cleared       string `csv:"Cleared"`
}
